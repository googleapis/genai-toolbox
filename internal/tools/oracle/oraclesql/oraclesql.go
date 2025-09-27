// Copyright 2024 Google LLC
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

package oraclesql

import (
        "context"
        "database/sql"
        "fmt"

        yaml "github.com/goccy/go-yaml"
        "github.com/googleapis/genai-toolbox/internal/sources"
        "github.com/googleapis/genai-toolbox/internal/sources/oracle"
        "github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "oracle-sql"

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

        allParameters, paramManifest, paramMcpManifest, err := tools.ProcessParameters(cfg.TemplateParameters, cfg.Parameters)
        if err != nil {
                return nil, fmt.Errorf("error processing parameters: %w", err)
        }

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
                DB:                 s.OracleDB(),
                manifest:           tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
                mcpManifest:        mcpManifest,
        }
        return t, nil
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

        DB          *sql.DB
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

        // Debug output to understand what's happening
        fmt.Printf("=== DEBUG for tool: %s ===\n", t.Name)
        fmt.Printf("Original SQL: '%s'\n", t.Statement)
        fmt.Printf("After template resolution: '%s'\n", newStatement)
        fmt.Printf("Parameter count: %d\n", len(sliceParams))
        fmt.Printf("Parameter values: %+v\n", sliceParams)
        fmt.Printf("Parameter types: ")
        for i, p := range sliceParams {
            fmt.Printf("[%d]=%T ", i, p)
        }
        fmt.Printf("\n")

        // NO PARAMETER CONVERSION - godror supports :1, :2, :3 natively
        // Execute Oracle query with original statement
        rows, err := t.DB.QueryContext(ctx, newStatement, sliceParams...)
        if err != nil {
                return nil, fmt.Errorf("unable to execute Oracle query: %w", err)
        }
        defer rows.Close()

        // Get column names
        columns, err := rows.Columns()
        if err != nil {
                return nil, fmt.Errorf("unable to get columns: %w", err)
        }

        var out []any
        for rows.Next() {
                // Create slice to hold values
                values := make([]interface{}, len(columns))
                valuePtrs := make([]interface{}, len(columns))
                for i := range values {
                        valuePtrs[i] = &values[i]
                }

                // Scan the values
                if err := rows.Scan(valuePtrs...); err != nil {
                        return nil, fmt.Errorf("unable to scan row: %w", err)
                }

                // Create result map
                vMap := make(map[string]any)
                for i, col := range columns {
                        val := values[i]
                        if b, ok := val.([]byte); ok {
                                vMap[col] = string(b)
                        } else {
                                vMap[col] = val
                        }
                }
                out = append(out, vMap)
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
