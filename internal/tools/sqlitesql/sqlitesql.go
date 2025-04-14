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
    "database/sql"
    "fmt"

    "github.com/googleapis/genai-toolbox/internal/sources"
    "github.com/googleapis/genai-toolbox/internal/sources/sqlite"
    "github.com/googleapis/genai-toolbox/internal/tools"
)

const ToolKind string = "sqlite-sql"

type compatibleSource interface {
    SQLiteDB() *sql.DB
}

// validate compatible sources are still compatible
var _ compatibleSource = &sqlite.Source{}

var compatibleSources = [...]string{sqlite.SourceKind}

type Config struct {
    Name         string           `yaml:"name" validate:"required"`
    Kind         string           `yaml:"kind" validate:"required"`
    Source       string           `yaml:"source" validate:"required"`
    Description  string           `yaml:"description" validate:"required"`
    Statement    string           `yaml:"statement" validate:"required"`
    AuthRequired []string         `yaml:"authRequired"`
    Parameters   tools.Parameters `yaml:"parameters"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
    return ToolKind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
    // verify source exists
    rawS, ok := srcs[cfg.Source]
    if (!ok) {
        return nil, fmt.Errorf("no source named %q configured", cfg.Source)
    }

    // verify the source is compatible
    s, ok := rawS.(compatibleSource)
    if (!ok) {
        return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", ToolKind, compatibleSources)
    }

    mcpManifest := tools.McpManifest{
        Name:        cfg.Name,
        Description: cfg.Description,
        InputSchema: cfg.Parameters.McpManifest(),
    }

    // finish tool setup
    t := Tool{
        Name:         cfg.Name,
        Kind:         ToolKind,
        Parameters:   cfg.Parameters,
        Statement:    cfg.Statement,
        AuthRequired: cfg.AuthRequired,
        Db:          s.SQLiteDB(),
        manifest:     tools.Manifest{Description: cfg.Description, Parameters: cfg.Parameters.Manifest()},
        mcpManifest: mcpManifest,
    }
    return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
    Name         string           `yaml:"name"`
    Kind         string           `yaml:"kind"`
    Parameters   tools.Parameters `yaml:"parameters"`
    Statement    string          `yaml:"statement"`
    AuthRequired []string        `yaml:"authRequired"`
    Db          *sql.DB
    manifest     tools.Manifest
    mcpManifest  tools.McpManifest
}

func (t Tool) ToolKind() string {
    return ToolKind
}

func (t Tool) GetManifest() tools.Manifest {
    return t.manifest
}

func (t Tool) GetMcpManifest() tools.McpManifest {
    return t.mcpManifest
}

func (t Tool) AuthenticationRequired() []string {
    return t.AuthRequired
}

func (t Tool) Invoke(params tools.ParamValues) ([]any, error) {
    // Execute the SQL query with parameters
    rows, err := t.Db.Query(t.Statement, params.AsSlice()...)
    if err != nil {
        return nil, fmt.Errorf("unable to execute query: %w", err)
    }
    defer rows.Close()

    // Get column names
    cols, err := rows.Columns()
    if err != nil {
        return nil, fmt.Errorf("unable to get column names: %w", err)
    }

    // Prepare the result slice
    var result []any

    // Iterate through the rows
    for rows.Next() {
        // Create a slice of interface{} to hold each row's values
        values := make([]interface{}, len(cols))
        valuePtrs := make([]interface{}, len(cols))
        for i := range values {
            valuePtrs[i] = &values[i]
        }

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

    if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating rows: %w", err)
    }

    return result, nil
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