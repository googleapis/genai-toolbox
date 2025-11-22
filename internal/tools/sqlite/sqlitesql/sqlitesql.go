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

package sqlitesql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/sqlite"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind string = "sqlite-sql"

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
	SQLiteDB() *sql.DB
}

// validate compatible sources are still compatible
var _ compatibleSource = &sqlite.Source{}

var compatibleSources = [...]string{sqlite.SourceKind}

type Config struct {
	Name               string                `yaml:"name" validate:"required"`
	Kind               string                `yaml:"kind" validate:"required"`
	Source             string                `yaml:"source" validate:"required"`
	Description        string                `yaml:"description" validate:"required"`
	Statement          string                `yaml:"statement" validate:"required"`
	AuthRequired       []string              `yaml:"authRequired"`
	Parameters         parameters.Parameters `yaml:"parameters"`
	TemplateParameters parameters.Parameters `yaml:"templateParameters"`
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

	allParameters, paramManifest, err := parameters.ProcessParameters(cfg.TemplateParameters, cfg.Parameters)
	if err != nil {
		return nil, err
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		AllParams:   allParameters,
		Db:          s.SQLiteDB(),
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	AllParams parameters.Parameters `yaml:"allParams"`

	Db          *sql.DB
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()
	newStatement, err := parameters.ResolveTemplateParams(t.TemplateParameters, t.Statement, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract template params %w", err)
	}

	newParams, err := parameters.GetParams(t.Parameters, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract standard params %w", err)
	}

	// Execute the SQL query with parameters
	rows, err := t.Db.QueryContext(ctx, newStatement, newParams.AsSlice()...)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("unable to get column names: %w", err)
	}

	// The sqlite driver does not support ColumnTypes, so we can't get the
	// underlying database type of the columns. We'll have to rely on the
	// generic `any` type and then handle the JSON data separately.
	rawValues := make([]any, len(cols))
	values := make([]any, len(cols))
	for i := range rawValues {
		values[i] = &rawValues[i]
	}

	// Prepare the result slice
	var out []any
	for rows.Next() {
		if err := rows.Scan(values...); err != nil {
			return nil, fmt.Errorf("unable to scan row: %w", err)
		}

		// Create a map for this row
		vMap := make(map[string]any)
		for i, name := range cols {
			val := rawValues[i]
			// Handle nil values
			if val == nil {
				vMap[name] = nil
				continue
			}
			// Handle JSON data
			if jsonString, ok := val.(string); ok {
				var unmarshaledData any
				if json.Unmarshal([]byte(jsonString), &unmarshaledData) == nil {
					vMap[name] = unmarshaledData
					continue
				}
			}
			// Store the value in the map
			vMap[name] = val
		}
		out = append(out, vMap)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return out, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.AllParams, data, claims)
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
