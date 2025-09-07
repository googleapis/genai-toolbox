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

package trinolisttables

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/trino"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "trino-list-tables"

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
		tools.NewStringParameter("catalog", "Optional: Catalog name. If not provided, uses the current catalog."),
		tools.NewStringParameter("schema", "Optional: Schema name. If not provided, uses the current schema."),
		tools.NewStringParameter("table_filter", "Optional: Filter tables by name pattern (supports SQL LIKE wildcards: % and _)."),
		tools.NewBooleanParameterWithDefault("include_views", true, "If true, includes views in the results. Default is true."),
		tools.NewBooleanParameterWithDefault("include_details", false, "If true, includes additional details like column count. Default is false."),
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

// TableInfo represents information about a table
type TableInfo struct {
	CatalogName string `json:"catalogName"`
	SchemaName  string `json:"schemaName"`
	TableName   string `json:"tableName"`
	TableType   string `json:"tableType"`
	ColumnCount int    `json:"columnCount,omitempty"`
	Comment     string `json:"comment,omitempty"`
}

// TablesResponse represents the response containing tables
type TablesResponse struct {
	Catalog    string      `json:"catalog"`
	Schema     string      `json:"schema"`
	Tables     []TableInfo `json:"tables"`
	TotalCount int         `json:"totalCount"`
	ViewCount  int         `json:"viewCount"`
	TableCount int         `json:"tableCount"`
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
	schema, _ := paramsMap["schema"].(string)
	tableFilter, _ := paramsMap["table_filter"].(string)
	includeViews, _ := paramsMap["include_views"].(bool)
	includeDetails, _ := paramsMap["include_details"].(bool)

	// Build the base query
	var queryBuilder strings.Builder
	var args []interface{}

	if includeDetails {
		queryBuilder.WriteString(`
			SELECT 
				t.table_catalog,
				t.table_schema,
				t.table_name,
				t.table_type,
				COUNT(c.column_name) as column_count
			FROM information_schema.tables t
			LEFT JOIN information_schema.columns c
				ON t.table_catalog = c.table_catalog
				AND t.table_schema = c.table_schema
				AND t.table_name = c.table_name
			WHERE 1=1
		`)
	} else {
		queryBuilder.WriteString(`
			SELECT 
				t.table_catalog,
				t.table_schema,
				t.table_name,
				t.table_type,
				0 as column_count
			FROM information_schema.tables t
			WHERE 1=1
		`)
	}

	// Add catalog filter
	if catalog == "" {
		queryBuilder.WriteString(` AND t.table_catalog = CURRENT_CATALOG`)
	} else {
		queryBuilder.WriteString(` AND t.table_catalog = ?`)
		args = append(args, catalog)
	}

	// Add schema filter
	if schema == "" {
		queryBuilder.WriteString(` AND t.table_schema = CURRENT_SCHEMA`)
	} else {
		queryBuilder.WriteString(` AND t.table_schema = ?`)
		args = append(args, schema)
	}

	// Add table type filter
	if includeViews {
		queryBuilder.WriteString(` AND t.table_type IN ('BASE TABLE', 'VIEW')`)
	} else {
		queryBuilder.WriteString(` AND t.table_type = 'BASE TABLE'`)
	}

	// Add table name filter if provided
	if tableFilter != "" {
		queryBuilder.WriteString(` AND t.table_name LIKE ?`)
		args = append(args, tableFilter)
	}

	// Add GROUP BY and ORDER BY
	if includeDetails {
		queryBuilder.WriteString(` GROUP BY t.table_catalog, t.table_schema, t.table_name, t.table_type`)
	}
	queryBuilder.WriteString(` ORDER BY t.table_type, t.table_name`)

	query := queryBuilder.String()
	rows, err := t.Db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	var tables []TableInfo
	var actualCatalog, actualSchema string
	viewCount := 0
	tableCount := 0

	for rows.Next() {
		var table TableInfo
		if err := rows.Scan(&table.CatalogName, &table.SchemaName, &table.TableName, &table.TableType, &table.ColumnCount); err != nil {
			return nil, fmt.Errorf("failed to scan table row: %w", err)
		}

		// Track actual catalog and schema from results
		if actualCatalog == "" {
			actualCatalog = table.CatalogName
		}
		if actualSchema == "" {
			actualSchema = table.SchemaName
		}

		// Count table types
		switch table.TableType {
		case "VIEW":
			viewCount++
		case "BASE TABLE":
			tableCount++
		}

		tables = append(tables, table)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating table rows: %w", err)
	}

	// Use actual values from results if we used CURRENT_CATALOG/CURRENT_SCHEMA
	if catalog == "" && actualCatalog != "" {
		catalog = actualCatalog
	}
	if schema == "" && actualSchema != "" {
		schema = actualSchema
	}

	// If still empty, query for current values
	if catalog == "" || schema == "" {
		var currentCatalog, currentSchema string
		err := t.Db.QueryRowContext(ctx, "SELECT CURRENT_CATALOG, CURRENT_SCHEMA").Scan(&currentCatalog, &currentSchema)
		if err == nil {
			if catalog == "" {
				catalog = currentCatalog
			}
			if schema == "" {
				schema = currentSchema
			}
		}
	}

	response := TablesResponse{
		Catalog:    catalog,
		Schema:     schema,
		Tables:     tables,
		TotalCount: len(tables),
		ViewCount:  viewCount,
		TableCount: tableCount,
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
