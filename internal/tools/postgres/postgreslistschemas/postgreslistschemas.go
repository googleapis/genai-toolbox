// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package postgreslistschemas

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/alloydbpg"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqlpg"
	"github.com/googleapis/genai-toolbox/internal/sources/postgres"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"github.com/jackc/pgx/v5/pgxpool"
)

const kind string = "postgres-list-schemas"

const listSchemasStatement = `
	WITH
	schema_grants AS (
		SELECT schema_oid, jsonb_object_agg(grantee, privileges) AS grants
		FROM
		(
			SELECT
			n.oid AS schema_oid,
			CASE
				WHEN p.grantee = 0 THEN 'PUBLIC'
				ELSE pg_catalog.pg_get_userbyid(p.grantee)
				END
				AS grantee,
			jsonb_agg(p.privilege_type ORDER BY p.privilege_type) AS privileges
			FROM pg_catalog.pg_namespace n, aclexplode(n.nspacl) p
			WHERE n.nspacl IS NOT NULL
			GROUP BY n.oid, grantee
		) permissions_by_grantee
		GROUP BY schema_oid
	),
	all_schemas AS (
		SELECT
		n.nspname AS schema_name,
		pg_catalog.pg_get_userbyid(n.nspowner) AS owner,
		COALESCE(sg.grants, '{}'::jsonb) AS grants,
		(
			SELECT COUNT(*)
			FROM pg_catalog.pg_class c
			WHERE c.relnamespace = n.oid AND c.relkind = 'r'
		) AS tables,
		(
			SELECT COUNT(*)
			FROM pg_catalog.pg_class c
			WHERE c.relnamespace = n.oid AND c.relkind = 'v'
		) AS views,
		(SELECT COUNT(*) FROM pg_catalog.pg_proc p WHERE p.pronamespace = n.oid)
			AS functions
		FROM pg_catalog.pg_namespace n
		LEFT JOIN schema_grants sg
		ON n.oid = sg.schema_oid
	)
	SELECT *
	FROM all_schemas
	-- Exclude system schemas and temporary schemas created per session.
	WHERE
	schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
	AND schema_name NOT LIKE 'pg_temp_%'
	AND ($1::text IS NULL OR schema_name LIKE '%' || $1::text || '%')
	ORDER BY schema_name;
`

func init() {
	if !tools.Register(kind, newConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", kind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type compatibleSource interface {
	PostgresPool() *pgxpool.Pool
}

// validate compatible sources are still compatible
var _ compatibleSource = &alloydbpg.Source{}
var _ compatibleSource = &cloudsqlpg.Source{}
var _ compatibleSource = &postgres.Source{}

var compatibleSources = [...]string{alloydbpg.SourceKind, cloudsqlpg.SourceKind, postgres.SourceKind}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	allParameters := parameters.Parameters{
		parameters.NewStringParameterWithDefault("schema_name", "", "Optional: A specific schema name pattern to search for."),
	}
	description := cfg.Description
	if description == "" {
		description = "Lists all schemas in the database ordered by schema name and excluding system and temporary schemas. It returns the schema name, schema owner, grants, number of functions, number of tables and number of views within each schema."
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, description, cfg.AuthRequired, allParameters, nil)

	// finish tool setup
	return Tool{
		Config:    cfg,
		allParams: allParameters,
		pool:      s.PostgresPool(),
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   allParameters.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	allParams   parameters.Parameters `yaml:"allParams"`
	pool        *pgxpool.Pool
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	sliceParams := params.AsSlice()

	results, err := t.pool.Query(ctx, listSchemasStatement, sliceParams...)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer results.Close()

	fields := results.FieldDescriptions()
	var out []map[string]any

	for results.Next() {
		values, err := results.Values()
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}
		rowMap := make(map[string]any)
		for i, field := range fields {
			rowMap[string(field.Name)] = values[i]
		}
		out = append(out, rowMap)
	}

	// this will catch actual query execution errors
	if err := results.Err(); err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}

	return out, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.allParams, data, claims)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) RequiresClientAuthorization() bool {
	return false
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
