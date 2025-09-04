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

package trinolistschemas

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/trino"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "trino-list-schemas"

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
	TrinoDB() *sql.DB
}

// validate compatible sources are still compatible
var _ compatibleSource = &trino.Source{}

var compatibleSources = [...]string{trino.SourceKind}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
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

	// Define parameters
	parameters := tools.Parameters{
		tools.NewStringParameter("catalog", "Optional: Catalog name to list schemas from. If not provided, uses the current catalog."),
		tools.NewBooleanParameterWithDefault("include_system", false, "If true, includes system schemas (information_schema, etc.). Default is false."),
	}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	// finish tool setup
	t := Tool{
		Name:         cfg.Name,
		Kind:         kind,
		Parameters:   parameters,
		AuthRequired: cfg.AuthRequired,
		Db:           s.TrinoDB(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

// SchemaInfo represents information about a schema
type SchemaInfo struct {
	CatalogName string `json:"catalogName"`
	SchemaName  string `json:"schemaName"`
	TableCount  int    `json:"tableCount,omitempty"`
	ViewCount   int    `json:"viewCount,omitempty"`
}

// SchemasResponse represents the response containing schemas
type SchemasResponse struct {
	Catalog    string       `json:"catalog"`
	Schemas    []SchemaInfo `json:"schemas"`
	TotalCount int          `json:"totalCount"`
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`

	Db          *sql.DB
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	catalog, _ := paramsMap["catalog"].(string)
	includeSystem, _ := paramsMap["include_system"].(bool)

	// Build the query
	var query string
	var args []interface{}

	if catalog == "" {
		// Use current catalog
		query = `
			SELECT 
				s.catalog_name,
				s.schema_name,
				COUNT(DISTINCT CASE WHEN t.table_type = 'BASE TABLE' THEN t.table_name END) as table_count,
				COUNT(DISTINCT CASE WHEN t.table_type = 'VIEW' THEN t.table_name END) as view_count
			FROM information_schema.schemata s
			LEFT JOIN information_schema.tables t 
				ON s.catalog_name = t.table_catalog 
				AND s.schema_name = t.table_schema
			WHERE s.catalog_name = CURRENT_CATALOG
		`
	} else {
		// Use specified catalog
		query = `
			SELECT 
				s.catalog_name,
				s.schema_name,
				COUNT(DISTINCT CASE WHEN t.table_type = 'BASE TABLE' THEN t.table_name END) as table_count,
				COUNT(DISTINCT CASE WHEN t.table_type = 'VIEW' THEN t.table_name END) as view_count
			FROM information_schema.schemata s
			LEFT JOIN information_schema.tables t 
				ON s.catalog_name = t.table_catalog 
				AND s.schema_name = t.table_schema
			WHERE s.catalog_name = ?
		`
		args = append(args, catalog)
	}

	// Filter out system schemas unless requested
	if !includeSystem {
		query += ` AND s.schema_name NOT IN ('information_schema', 'pg_catalog', 'sys')`
	}

	query += ` GROUP BY s.catalog_name, s.schema_name ORDER BY s.schema_name`

	rows, err := t.Db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}
	defer rows.Close()

	var schemas []SchemaInfo
	var actualCatalog string

	for rows.Next() {
		var schema SchemaInfo
		if err := rows.Scan(&schema.CatalogName, &schema.SchemaName, &schema.TableCount, &schema.ViewCount); err != nil {
			return nil, fmt.Errorf("failed to scan schema row: %w", err)
		}
		if actualCatalog == "" {
			actualCatalog = schema.CatalogName
		}
		schemas = append(schemas, schema)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schema rows: %w", err)
	}

	// If we used CURRENT_CATALOG and got results, use the actual catalog name
	if catalog == "" && actualCatalog != "" {
		catalog = actualCatalog
	} else if catalog == "" {
		// If still empty, get the current catalog
		var currentCatalog string
		err := t.Db.QueryRowContext(ctx, "SELECT CURRENT_CATALOG").Scan(&currentCatalog)
		if err == nil {
			catalog = currentCatalog
		}
	}

	response := SchemasResponse{
		Catalog:    catalog,
		Schemas:    schemas,
		TotalCount: len(schemas),
	}

	return response, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data, claims)
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
