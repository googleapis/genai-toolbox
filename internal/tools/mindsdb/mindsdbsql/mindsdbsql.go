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

package mindsdbsql

import (
	"context"
	"database/sql"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/mindsdb"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/mysql/mysqlcommon"
)

const kind string = "mindsdb-sql"

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
	MindsDBPool() *sql.DB
}

// validate compatible sources are still compatible
var _ compatibleSource = &mindsdb.Source{}

var compatibleSources = [...]string{mindsdb.SourceKind}

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

	allParameters, paramManifest, err := tools.ProcessParameters(cfg.TemplateParameters, cfg.Parameters)
	if err != nil {
		return nil, err
	}

	paramMcpManifest, _ := allParameters.McpManifest()

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: paramMcpManifest,
	}

	// finish tool setup
	t := Tool{
		Name:               cfg.Name,
		Kind:               kind,
		Parameters:         cfg.Parameters,
		TemplateParameters: cfg.TemplateParameters,
		AllParams:          allParameters,
		Statement:          cfg.Statement,
		AuthRequired:       cfg.AuthRequired,
		Pool:               s.MindsDBPool(),
		manifest:           tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:        mcpManifest,
	}
	return t, nil
}

// interpolateParams replaces ? placeholders with actual parameter values
// This is necessary because MindsDB doesn't support MySQL prepared statements
func interpolateParams(query string, params []any) (string, error) {
	result := query
	paramIndex := 0

	for paramIndex < len(params) {
		// Find the next ? placeholder
		idx := -1
		for i, ch := range result {
			if ch == '?' {
				idx = i
				break
			}
		}
		if idx == -1 {
			break // No more placeholders
		}

		param := params[paramIndex]
		var replacement string

		switch v := param.(type) {
		case nil:
			replacement = "NULL"
		case string:
			// Escape single quotes in strings
			escaped := ""
			for _, ch := range v {
				if ch == '\'' {
					escaped += "''"
				} else {
					escaped += string(ch)
				}
			}
			replacement = "'" + escaped + "'"
		case int, int8, int16, int32, int64:
			replacement = fmt.Sprintf("%d", v)
		case uint, uint8, uint16, uint32, uint64:
			replacement = fmt.Sprintf("%d", v)
		case float32, float64:
			replacement = fmt.Sprintf("%v", v)
		case bool:
			if v {
				replacement = "1"
			} else {
				replacement = "0"
			}
		default:
			// For other types, try string conversion
			str := fmt.Sprintf("%v", v)
			escaped := ""
			for _, ch := range str {
				if ch == '\'' {
					escaped += "''"
				} else {
					escaped += string(ch)
				}
			}
			replacement = "'" + escaped + "'"
		}

		// Replace the first ? with the parameter value
		result = result[:idx] + replacement + result[idx+1:]
		paramIndex++
	}

	return result, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name               string           `yaml:"name"`
	Kind               string           `yaml:"kind"`
	AuthRequired       []string         `yaml:"authRequired"`
	Parameters         tools.Parameters `yaml:"parameters"`
	TemplateParameters tools.Parameters `yaml:"templateParameters"`
	AllParams          tools.Parameters `yaml:"allParams"`

	Pool        *sql.DB
	Statement   string
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
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

	// MindsDB doesn't support prepared statements, so we need to interpolate parameters manually
	// Replace ? placeholders with actual values
	finalStatement, err := interpolateParams(newStatement, sliceParams)
	if err != nil {
		return nil, fmt.Errorf("unable to interpolate params: %w", err)
	}

	results, err := t.Pool.QueryContext(ctx, finalStatement)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}

	cols, err := results.Columns()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve rows column name: %w", err)
	}

	// create an array of values for each column, which can be re-used to scan each row
	rawValues := make([]any, len(cols))
	values := make([]any, len(cols))
	for i := range rawValues {
		values[i] = &rawValues[i]
	}
	defer results.Close()

	colTypes, err := results.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("unable to get column types: %w", err)
	}

	var out []any
	for results.Next() {
		err := results.Scan(values...)
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}
		vMap := make(map[string]any)
		for i, name := range cols {
			val := rawValues[i]
			if val == nil {
				vMap[name] = nil
				continue
			}

			// MindsDB uses mysql driver
			vMap[name], err = mysqlcommon.ConvertToType(colTypes[i], val)
			if err != nil {
				return nil, fmt.Errorf("errors encountered when converting values: %w", err)
			}
		}
		out = append(out, vMap)
	}

	if err := results.Err(); err != nil {
		return nil, fmt.Errorf("errors encountered during row iteration: %w", err)
	}

	return out, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.AllParams, data, claims)
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
