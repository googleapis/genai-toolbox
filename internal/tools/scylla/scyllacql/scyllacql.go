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

package scyllacql

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/gocql/gocql"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/scylla"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "scylla-cql"

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
	ScyllaSession() *gocql.Session
}

// validate compatible sources are still compatible
var _ compatibleSource = &scylla.Source{}

var compatibleSources = [...]string{scylla.SourceKind}

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
		return nil, fmt.Errorf("unable to process parameters: %w", err)
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
		Session:            s.ScyllaSession(),
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

	Statement   string
	Session     *gocql.Session
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

	// Execute the query with parameters
	iter := t.Session.Query(newStatement, sliceParams...).Iter()
	defer iter.Close()

	// Get column information
	columns := iter.Columns()
	if len(columns) == 0 {
		// This might be a non-SELECT query (INSERT, UPDATE, DELETE, etc.)
		// Check if there was an error
		if err := iter.Close(); err != nil {
			return nil, fmt.Errorf("unable to execute query: %w", err)
		}
		return map[string]string{"status": "success", "message": "Query executed successfully"}, nil
	}

	// Create a map to store row data
	var results []map[string]any

	// Iterate over the results
	for {
		// Create a map for this row
		row := make(map[string]any)

		// Create a slice to hold the values for scanning
		values := make([]any, len(columns))
		for i := range values {
			values[i] = new(any)
		}

		// Scan the row
		if !iter.Scan(values...) {
			break
		}

		// Populate the row map
		for i, col := range columns {
			val := *(values[i].(*any))

			// Convert []uint8 to string for better readability
			if b, ok := val.([]uint8); ok {
				row[col.Name] = string(b)
			} else {
				row[col.Name] = val
			}
		}

		results = append(results, row)
	}

	// Check for any errors during iteration
	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("errors encountered during query execution: %w", err)
	}

	return results, nil
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
