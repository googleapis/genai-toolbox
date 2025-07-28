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

package bigqueryexecutesql

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	bigqueryapi "cloud.google.com/go/bigquery"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	bigqueryds "github.com/googleapis/genai-toolbox/internal/sources/bigquery"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"google.golang.org/api/iterator"
)

const kind string = "bigquery-execute-sql"

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
	BigQueryClient() *bigqueryapi.Client
	ValidateDatasetAccess(dataset string) error
}

// validate compatible sources are still compatible
var _ compatibleSource = &bigqueryds.Source{}

var compatibleSources = [...]string{bigqueryds.SourceKind}

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

	sqlParameter := tools.NewStringParameter("sql", "The sql to execute.")
	parameters := tools.Parameters{sqlParameter}

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
		Client:       s.BigQueryClient(),
		Source:       s,
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
	Client       *bigqueryapi.Client
	Source       compatibleSource // Source for dataset validation
	manifest     tools.Manifest
	mcpManifest  tools.McpManifest
}

func (t Tool) ValidateSQLDatasetAccess(sql string) error {
	// Simple approach: find all backtick-quoted table references
	// Pattern: `something.something` or `something.something.something`
	backtickPattern := regexp.MustCompile(`\x60([^\x60]+)\x60`)
	backtickMatches := backtickPattern.FindAllStringSubmatch(sql, -1)

	var datasets []string

	// Extract datasets from backtick matches
	for _, match := range backtickMatches {
		if len(match) > 1 {
			tableRef := match[1] // Content inside backticks
			parts := strings.Split(tableRef, ".")

			var dataset string
			if len(parts) == 2 { // dataset.table
				dataset = parts[0]
			} else if len(parts) == 3 { // project.dataset.table
				dataset = parts[1]
			}

			if dataset != "" {
				datasets = append(datasets, dataset)
			}
		}
	}

	// Also check for non-backtick table references
	noBacktickPattern := regexp.MustCompile(`(?i)\bFROM\s+([a-zA-Z_][a-zA-Z0-9_]*\.[a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)\b`)
	noBacktickMatches := noBacktickPattern.FindAllStringSubmatch(sql, -1)

	for _, match := range noBacktickMatches {
		if len(match) > 1 {
			tableRef := match[1]
			parts := strings.Split(tableRef, ".")

			var dataset string
			if len(parts) == 2 { // dataset.table
				dataset = parts[0]
			} else if len(parts) == 3 { // project.dataset.table
				dataset = parts[1]
			}

			if dataset != "" {
				datasets = append(datasets, dataset)
			}
		}
	}

	if len(datasets) == 0 {
		return fmt.Errorf("no valid table references found in SQL query")
	}

	// Use source-level validation for each dataset found
	for _, dataset := range datasets {
		if err := t.Source.ValidateDatasetAccess(dataset); err != nil {
			return err
		}
	}

	// Ensure query has LIMIT clause (add if missing)
	if !strings.Contains(strings.ToUpper(sql), "LIMIT") {
		// This is just a warning - the actual LIMIT enforcement should be done by modifying the query
		// For now, we'll just validate that queries should have LIMIT
		return fmt.Errorf("query must include LIMIT clause (max 1000 rows)")
	}

	return nil
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	sliceParams := params.AsSlice()
	sql, ok := sliceParams[0].(string)
	if !ok {
		return nil, fmt.Errorf("unable to get cast %s", sliceParams[0])
	}

	// Extract datasets from SQL and validate access using source-level validation
	if err := t.ValidateSQLDatasetAccess(sql); err != nil {
		return nil, fmt.Errorf("SQL query violates dataset restrictions: %w", err)
	}

	query := t.Client.Query(sql)
	query.Location = t.Client.Location

	it, err := query.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}

	var out []any
	for {
		var row map[string]bigqueryapi.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("unable to iterate through query results: %w", err)
		}
		vMap := make(map[string]any)
		for key, value := range row {
			vMap[key] = value
		}
		out = append(out, vMap)
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
