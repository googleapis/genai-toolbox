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

package duckdbsql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/duckdb"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "duckdb-sql"

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
	DuckDb() *sql.DB
}

// validate compatible sources are still compatible
var _ compatibleSource = &duckdb.Source{}
var compatibleSources = [...]string{duckdb.SourceKind}

type Config struct {
	Name               string           `yaml:"name" validate:"required"`
	Kind               string           `yaml:"kind" validate:"required"`
	Source             string           `yaml:"source" validate:"required"`
	Description        string           `yaml:"description" validate:"required"`
	Statement          string           `yaml:"statement" validate:"required"`
	AuthRequired       []string         `yaml:"authRequired"`
	Parameters         tools.Parameters `yaml:"parameters"`
	TemplateParameters tools.Parameters `yaml:"templateParameters"`
}

// Initialize implements tools.ToolConfig.
func (c Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[c.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", c.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	allParameters, paramManifest, paramMcpManifest := tools.ProcessParameters(c.TemplateParameters, c.Parameters)

	mcpManifest := tools.McpManifest{
		Name:        c.Name,
		Description: c.Description,
		InputSchema: paramMcpManifest,
	}

	// finish tool setup
	t := Tool{
		Name:               c.Name,
		Kind:               kind,
		Parameters:         c.Parameters,
		TemplateParameters: c.TemplateParameters,
		AllParams:          allParameters,
		Statement:          c.Statement,
		AuthRequired:       c.AuthRequired,
		Db:                 s.DuckDb(),
		manifest:           tools.Manifest{Description: c.Description, Parameters: paramManifest, AuthRequired: c.AuthRequired},
		mcpManifest:        mcpManifest,
	}
	return t, nil
}

// ToolConfigKind implements tools.ToolConfig.
func (c Config) ToolConfigKind() string {
	return kind
}

var _ tools.ToolConfig = Config{}

type Tool struct {
	Name               string           `yaml:"name"`
	Kind               string           `yaml:"kind"`
	AuthRequired       []string         `yaml:"authRequired"`
	Parameters         tools.Parameters `yaml:"parameters"`
	TemplateParameters tools.Parameters `yaml:"templateParameters"`
	AllParams          tools.Parameters `yaml:"allParams"`

	Db          *sql.DB
	Statement   string `yaml:"statement"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// Authorized implements tools.Tool.
func (t Tool) Authorized(verifiedAuthSources []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthSources)
}

// Invoke implements tools.Tool.
func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accesToken tools.OAuthAccessToken) (any, error) {
	paramsMap := params.AsMap()
	newStatement, err := tools.ResolveTemplateParams(t.TemplateParameters, t.Statement, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract template params %w", err)
	}

	newParams, err := tools.GetParams(t.Parameters, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract standard params %w", err)
	}

	sliceParams := newParams.AsSlice()
	// Execute the SQL query with parameters
	rows, err := t.Db.QueryContext(ctx, newStatement, sliceParams...)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("unable to get column names: %w", err)
	}

	values := make([]any, len(cols))
	valuePtrs := make([]any, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Prepare the result slice
	var result []any
	// Iterate through the rows
	for rows.Next() {
		// Scan the row into the value pointers
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("unable to scan row: %w", err)
		}

		// Create a map for this row
		rowMap := make(map[string]interface{})
		for i, col := range cols {
			val := values[i]
			// Handle nil values
			if val == nil {
				rowMap[col] = nil
				continue
			}
			// Store the value in the map
			rowMap[col] = val
		}
		result = append(result, rowMap)
	}

	if err = rows.Close(); err != nil {
		return nil, fmt.Errorf("unable to close rows: %w", err)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// Manifest implements tools.Tool.
func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

// McpManifest implements tools.Tool.
func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

// ParseParams implements tools.Tool.
func (t Tool) ParseParams(data map[string]any, claimsMap map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.AllParams, data, claimsMap)
}

var _ tools.Tool = Tool{}
