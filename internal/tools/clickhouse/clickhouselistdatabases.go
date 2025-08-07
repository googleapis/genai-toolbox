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

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const listDatabasesKind string = "clickhouse-list-databases"

func init() {
	if !tools.Register(listDatabasesKind, newListDatabasesConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", listDatabasesKind))
	}
}

func newListDatabasesConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := ListDatabasesConfig{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type ListDatabasesConfig struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

var _ tools.ToolConfig = ListDatabasesConfig{}

func (cfg ListDatabasesConfig) ToolConfigKind() string {
	return listDatabasesKind
}

func (cfg ListDatabasesConfig) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", listDatabasesKind, compatibleSources)
	}

	parameters := tools.Parameters{}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	t := ListDatabasesTool{
		Name:         cfg.Name,
		Kind:         listDatabasesKind,
		Parameters:   parameters,
		AuthRequired: cfg.AuthRequired,
		Pool:         s.ClickHousePool(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

var _ tools.Tool = ListDatabasesTool{}

type ListDatabasesTool struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`

	Pool        *sql.DB
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t ListDatabasesTool) Invoke(ctx context.Context, params tools.ParamValues) (any, error) {
	query := `
		SELECT
			name as database_name,
			toJSONString(map(
				'database_name', name,
				'engine', engine,
				'comment', comment
			)) AS database_details
		FROM system.databases
		ORDER BY name`

	results, err := t.Pool.QueryContext(ctx, query)
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

func (t ListDatabasesTool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data, claims)
}

func (t ListDatabasesTool) Manifest() tools.Manifest {
	return t.manifest
}

func (t ListDatabasesTool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t ListDatabasesTool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}
