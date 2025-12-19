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

package postgreslistpublicationtables

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

const kind string = "postgres-list-publication-tables"

const listPublicationTablesStatement = `
	WITH
	publication_details AS (
		SELECT
			pt.pubname AS publication_name,
			pt.schemaname AS schema_name,
			pt.tablename AS table_name,
			-- Definition details
			p.puballtables AS publishes_all_tables,
			p.pubinsert AS publishes_inserts,
			p.pubupdate AS publishes_updates,
			p.pubdelete AS publishes_deletes,
			p.pubtruncate AS publishes_truncates,
			-- Owner information
			pg_catalog.pg_get_userbyid(p.pubowner) AS publication_owner
		FROM pg_catalog.pg_publication_tables pt
		JOIN pg_catalog.pg_publication p
			ON pt.pubname = p.pubname
	)
	SELECT *
	FROM publication_details
	WHERE
		(NULLIF(TRIM($1::text), '') IS NULL OR table_name = ANY(regexp_split_to_array(TRIM($1::text), '\s*,\s*')))
    	AND (NULLIF(TRIM($2::text), '') IS NULL OR publication_name = ANY(regexp_split_to_array(TRIM($2::text), '\s*,\s*')))
	    AND (NULLIF(TRIM($3::text), '') IS NULL OR schema_name = ANY(regexp_split_to_array(TRIM($3::text), '\s*,\s*')))
	ORDER BY
		publication_name, schema_name, table_name
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
		parameters.NewStringParameterWithDefault("table_names", "", "Optional: Filters by a comma-separated list of table names."),
		parameters.NewStringParameterWithDefault("publication_names", "", "Optional: Filters by a comma-separated list of publication names."),
		parameters.NewStringParameterWithDefault("schema_names", "", "Optional: Filters by a comma-separated list of schema names."),
		parameters.NewIntParameterWithDefault("limit", 50, "Optional: The maximum number of rows to return."),
	}
	description := cfg.Description
	if description == "" {
		description = "Lists all publication tables in the database. Returns the publication name, schema name, and table name, along with definition details indicating if it publishes all tables, whether it replicates inserts, updates, deletes, or truncates, and the publication owner."
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

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	newParams, err := parameters.GetParams(t.allParams, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract standard params %w", err)
	}
	sliceParams := newParams.AsSlice()

	results, err := t.pool.Query(ctx, listPublicationTablesStatement, sliceParams...)
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

	if err := results.Err(); err != nil {
		return err.Error(), fmt.Errorf("unable to execute query: %w", err)
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

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) bool {
	return false
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
