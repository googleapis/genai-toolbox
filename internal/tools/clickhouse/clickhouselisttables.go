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

package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const listTablesKind string = "clickhouse-list-tables"

func init() {
	if !tools.Register(listTablesKind, newListTablesConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", listTablesKind))
	}
}

func newListTablesConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := ListTablesConfig{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type ListTablesConfig struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

var _ tools.ToolConfig = ListTablesConfig{}

func (cfg ListTablesConfig) ToolConfigKind() string {
	return listTablesKind
}

func (cfg ListTablesConfig) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", listTablesKind, compatibleSources)
	}

	tableNamesParameter := tools.NewStringParameter("table_names", "Optional: A comma-separated list of table names. If empty, details for all tables in the current database will be listed.")
	parameters := tools.Parameters{tableNamesParameter}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	t := ListTablesTool{
		Name:         cfg.Name,
		Kind:         listTablesKind,
		Parameters:   parameters,
		AuthRequired: cfg.AuthRequired,
		Pool:         s.ClickHousePool(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

var _ tools.Tool = ListTablesTool{}

type ListTablesTool struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`

	Pool        *sql.DB
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t ListTablesTool) Invoke(ctx context.Context, params tools.ParamValues) (any, error) {
	sliceParams := params.AsSlice()
	tableNamesParam, ok := sliceParams[0].(string)
	if !ok {
		return nil, fmt.Errorf("unable to cast table_names parameter %v", sliceParams[0])
	}

	var query string
	var args []interface{}

	if strings.TrimSpace(tableNamesParam) == "" {
		query = `
			SELECT
				database as schema_name,
				name as object_name,
				toJSONString(map(
					'schema_name', database,
					'object_name', name,
					'object_type', 'TABLE',
					'engine', engine,
					'primary_key', primary_key,
					'sorting_key', sorting_key,
					'partition_key', partition_key,
					'total_rows', toString(total_rows),
					'total_bytes', toString(total_bytes),
					'comment', comment
				)) AS object_details
			FROM system.tables
			WHERE database = currentDatabase()
			ORDER BY name`
	} else {
		tableNames := strings.Split(tableNamesParam, ",")
		var trimmedNames []string
		var placeholders []string

		for _, name := range tableNames {
			trimmed := strings.TrimSpace(name)
			if trimmed != "" {
				trimmedNames = append(trimmedNames, trimmed)
				placeholders = append(placeholders, "?")
				args = append(args, trimmed)
			}
		}

		if len(trimmedNames) == 0 {
			return t.Invoke(ctx, tools.ParamValues{tools.ParamValue{Value: ""}})
		}

		inClause := strings.Join(placeholders, ", ")

		query = fmt.Sprintf(`
			SELECT
				database as schema_name,
				name as object_name,
				toJSONString(map(
					'schema_name', database,
					'object_name', name,
					'object_type', 'TABLE',
					'engine', engine,
					'primary_key', primary_key,
					'sorting_key', sorting_key,
					'partition_key', partition_key,
					'total_rows', toString(total_rows),
					'total_bytes', toString(total_bytes),
					'comment', comment
				)) AS object_details
			FROM system.tables
			WHERE database = currentDatabase()
			  AND name IN (%s)
			ORDER BY name`, inClause)
	}

	results, err := t.Pool.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer results.Close()

	cols, err := results.Columns()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve rows column name: %w", err)
	}

	rawValues := make([]any, len(cols))
	values := make([]any, len(cols))
	for i := range rawValues {
		values[i] = &rawValues[i]
	}

	var out []any
	for results.Next() {
		err := results.Scan(values...)
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}
		vMap := make(map[string]any)
		for i, name := range cols {
			vMap[name] = rawValues[i]
		}
		out = append(out, vMap)
	}

	if err := results.Err(); err != nil {
		return nil, fmt.Errorf("errors encountered by results.Scan: %w", err)
	}

	return out, nil
}

func (t ListTablesTool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data, claims)
}

func (t ListTablesTool) Manifest() tools.Manifest {
	return t.manifest
}

func (t ListTablesTool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t ListTablesTool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}
