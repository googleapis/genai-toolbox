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
// Copyright © 2025, Oracle and/or its affiliates.

package oracleexecutesql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/godror/godror"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/oracle"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
)

const kind string = "oracle-execute-sql"

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
	OracleDB() *sql.DB
}

// validate compatible sources are still compatible
var _ compatibleSource = &oracle.Source{}

var compatibleSources = [...]string{oracle.SourceKind}

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

	sqlParameter := tools.NewStringParameter("sql", "The SQL to execute.")
	parameters := tools.Parameters{sqlParameter}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, parameters)

	// finish tool setup
	t := Tool{
		Name:         cfg.Name,
		Kind:         kind,
		Parameters:   parameters,
		AuthRequired: cfg.AuthRequired,
		Pool:         s.OracleDB(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`

	Pool        *sql.DB
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()
	sql, ok := paramsMap["sql"].(string)
	if !ok {
		return nil, fmt.Errorf("unable to get cast %s", paramsMap["sql"])
	}

	// Log the query executed for debugging.
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting logger: %s", err)
	}
	logger.DebugContext(ctx, "executing `%s` tool query: %s", kind, sql)

	results, err := t.Pool.QueryContext(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer results.Close()

	cols, _ := results.Columns()
	// If Columns() errors, it might be a DDL/DML without an OUTPUT clause.
	// We proceed, and results.Err() will catch actual query execution errors.
	// 'out' will remain nil if cols is empty or err is not nil here.

	// Get Column types
	colTypes, err := results.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("unable to get column types: %w", err)
	}

	var out []any
	for results.Next() {
		// Create slice to hold values
		values := make([]any, len(cols))
		valuePtrs := make([]any, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the values
		if err := results.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("unable to scan row: %w", err)
		}

		// Create result map
		vMap := make(map[string]any)
		for i, col := range cols {
			val := values[i]
			switch colTypes[i].DatabaseTypeName() {
			case "JSON":
				// unmarshal JSON data before storing to prevent double marshaling
				var unmarshaledData any
				err := json.Unmarshal(val.([]byte), &unmarshaledData)
				if err != nil {
					return nil, fmt.Errorf("unable to unmarshal json data %s", val)
				}
				vMap[col] = unmarshaledData
			case "TEXT", "VARCHAR", "NVARCHAR":
				vMap[col] = string(val.([]byte))
			case "NUMBER":
				s := string(val.(godror.Number))
				if strings.Contains(s, ".") {
					vMap[col], err = strconv.ParseFloat(s, 64)
				} else {
					vMap[col], err = strconv.ParseInt(s, 10, 64)
				}
				if err != nil {
					return nil, fmt.Errorf("unable to convert NUMBER data '%s' for column %s: %w", s, col, err)
				}
			default:
				vMap[col] = val
			}
		}
		out = append(out, vMap)
	}

	// Check for errors from iterating over rows or from the query execution itself.
	// results.Close() is handled by defer.
	if err := results.Err(); err != nil {
		return nil, fmt.Errorf("errors encountered during query execution or row processing: %w", err)
	}

	return out, nil
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

func (t Tool) RequiresClientAuthorization() bool {
	return false
}
