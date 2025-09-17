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

package adxquery

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-kusto-go/kusto"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	adxds "github.com/googleapis/genai-toolbox/internal/sources/adx"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "adx-query"

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
	KustoClient() *kusto.Client
	GetDatabase() string
	ExecuteQuery(ctx context.Context, query string) ([]map[string]interface{}, error)
}

// validate compatible sources are still compatible
var _ compatibleSource = &adxds.Source{}

var compatibleSources = [...]string{adxds.SourceKind}

type Config struct {
	Name               string           `yaml:"name" validate:"required"`
	Kind               string           `yaml:"kind" validate:"required"`
	Source             string           `yaml:"source" validate:"required"`
	Description        string           `yaml:"description" validate:"required"`
	Query              string           `yaml:"query" validate:"required"`
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
	src, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("source %q not found", cfg.Source)
	}

	// verify source is compatible
	compatibleSource, ok := src.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("source %q (kind: %q) is not compatible with tool %q, compatible sources: %v", cfg.Source, src.SourceKind(), kind, compatibleSources)
	}

	allParameters, paramManifest, paramMcpManifest, err := tools.ProcessParameters(cfg.TemplateParameters, cfg.Parameters)
	if err != nil {
		return nil, err
	}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: paramMcpManifest,
	}

	return &Tool{
		Name:               cfg.Name,
		Kind:               kind,
		AuthRequired:       cfg.AuthRequired,
		Parameters:         cfg.Parameters,
		TemplateParameters: cfg.TemplateParameters,
		AllParams:          allParameters,
		Query:              cfg.Query,
		source:             compatibleSource,
		manifest:           tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:        mcpManifest,
	}, nil
}

type Tool struct {
	Name               string           `yaml:"name"`
	Kind               string           `yaml:"kind"`
	AuthRequired       []string         `yaml:"authRequired"`
	Parameters         tools.Parameters `yaml:"parameters"`
	TemplateParameters tools.Parameters `yaml:"templateParameters"`
	AllParams          tools.Parameters `yaml:"allParams"`
	Query              string
	source             compatibleSource
	manifest           tools.Manifest
	mcpManifest        tools.McpManifest
}

// validate interface
var _ tools.Tool = &Tool{}

func (t *Tool) ToolKind() string {
	return kind
}

func (t *Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	// Resolve template parameters in the query using Go template syntax (@paramName)
	query, err := tools.ResolveTemplateParams(t.TemplateParameters, t.Query, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve template params: %w", err)
	}

	// Handle positional parameters ($1, $2, etc.) by substituting them with parameter values in order
	// This allows users to use KQL queries like: "UsersInfo | where UserPrincipalName contains $1"
	paramValues := params.AsSlice()
	for i, value := range paramValues {
		placeholder := fmt.Sprintf("$%d", i+1)
		// Convert value to string for substitution
		var valueStr string
		if value == nil {
			valueStr = ""
		} else {
			switch v := value.(type) {
			case string:
				// For strings, wrap in quotes for KQL string literals
				valueStr = fmt.Sprintf("\"%s\"", v)
			case int, int32, int64, float32, float64:
				// For numbers, use as-is without quotes
				valueStr = fmt.Sprintf("%v", v)
			case bool:
				// For booleans, use lowercase string representation
				if v {
					valueStr = "true"
				} else {
					valueStr = "false"
				}
			default:
				// For other types, convert to string and wrap in quotes
				valueStr = fmt.Sprintf("\"%v\"", v)
			}
		}
		// Replace the placeholder with the actual value
		query = strings.ReplaceAll(query, placeholder, valueStr)
	}

	// Execute the query
	results, err := t.source.ExecuteQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute ADX query: %w", err)
	}

	return results, nil
}

func (t *Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.AllParams, data, claims)
}

func (t *Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t *Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t *Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t *Tool) RequiresClientAuthorization() bool {
	return false // ADX does not require client OAuth for now
}
