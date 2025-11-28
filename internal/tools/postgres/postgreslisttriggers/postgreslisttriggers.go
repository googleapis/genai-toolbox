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

package postgreslisttriggers

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

const kind string = "postgres-list-triggers"

const listTriggersStatement = `
	WITH
	trigger_list AS (
		SELECT
			t.tgname AS trigger_name,
			n.nspname AS schema_name,
			c.relname AS table_name,
			CASE t.tgenabled
				WHEN 'O' THEN 'ENABLED'
				WHEN 'D' THEN 'DISABLED'
				WHEN 'R' THEN 'REPLICA'
				WHEN 'A' THEN 'ALWAYS'
				END AS status,
			CASE
				WHEN (t.tgtype::int & 2) = 2 THEN 'BEFORE'
				WHEN (t.tgtype::int & 64) = 64 THEN 'INSTEAD OF'
				ELSE 'AFTER'
				END AS timing,
			concat_ws(
				', ',
				CASE WHEN (t.tgtype::int & 4) = 4 THEN 'INSERT' END,
				CASE WHEN (t.tgtype::int & 16) = 16 THEN 'UPDATE' END,
				CASE WHEN (t.tgtype::int & 8) = 8 THEN 'DELETE' END,
				CASE WHEN (t.tgtype::int & 32) = 32 THEN 'TRUNCATE' END) AS events,
			CASE WHEN (t.tgtype::int & 1) = 1 THEN 'ROW' ELSE 'STATEMENT' END AS activation_level,
			p.proname AS function_name,
			pg_get_triggerdef(t.oid) AS definition
		FROM pg_trigger t
		JOIN pg_class c
			ON t.tgrelid = c.oid
		JOIN pg_namespace n
			ON c.relnamespace = n.oid
		LEFT JOIN pg_proc p
			ON t.tgfoid = p.oid
		WHERE NOT t.tgisinternal
	)
	SELECT *
	FROM trigger_list
	WHERE
		($1::text IS NULL OR trigger_name LIKE '%' || $1::text || '%')
		AND ($2::text IS NULL OR schema_name LIKE '%' || $2::text || '%')
		AND ($3::text IS NULL OR table_name LIKE '%' || $3::text || '%')
	ORDER BY schema_name, table_name, trigger_name
	LIMIT COALESCE($4::int, 50);
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
		parameters.NewStringParameterWithDefault("trigger_name", "", "Optional: A specific trigger name pattern to search for."),
		parameters.NewStringParameterWithDefault("schema_name", "", "Optional: A specific schema name pattern to search for."),
		parameters.NewStringParameterWithDefault("table_name", "", "Optional: A specific table name pattern to search for."),
		parameters.NewIntParameterWithDefault("limit", 50, "Optional: The maximum number of rows to return."),
	}
	description := cfg.Description
	if description == "" {
		description = "Lists all non-internal triggers in a database. Returns trigger name, schema name, table name, whether its enabled or disabled, timing (e.g BEFORE/AFTER of the event), the  events that cause the trigger to fire such as INSERT, UPDATE, or DELETE, whether the trigger activates per ROW or per STATEMENT, the handler function executed by the trigger and full definition."
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

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	sliceParams := params.AsSlice()

	results, err := t.pool.Query(ctx, listTriggersStatement, sliceParams...)
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

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
