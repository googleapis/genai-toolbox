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

package kuzucypher

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	kuzuSource "github.com/googleapis/genai-toolbox/internal/sources/kuzu"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/kuzudb/go-kuzu"
)

var kind string = "kuzu-cypher"

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

type compatibleSource interface {
	KuzuDB() *kuzu.Connection
}

// validate compatible sources are still compatible
var _ compatibleSource = &kuzuSource.Source{}
var compatibleSources = [...]string{kuzuSource.SourceKind}

// Initialize implements tools.ToolConfig.
func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}
	allParameters, paramManifest, paramMcpManifest := tools.ProcessParameters(cfg.TemplateParameters, cfg.Parameters)
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
		Connection:         s.KuzuDB(),
		manifest:           tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:        mcpManifest,
	}
	return t, nil
}

// ToolConfigKind implements tools.ToolConfig.
func (cfg Config) ToolConfigKind() string {
	return kind
}

var _ tools.ToolConfig = Config{}

type Tool struct {
	Name               string           `yaml:"name" validate:"required"`
	Kind               string           `yaml:"kind"`
	AuthRequired       []string         `yaml:"authRequired"`
	Parameters         tools.Parameters `yaml:"parameters"`
	TemplateParameters tools.Parameters `yaml:"templateParameters"`
	AllParams          tools.Parameters `yaml:"allParams"`

	Connection  *kuzu.Connection
	Statement   string `yaml:"statement"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// Authorized implements tools.Tool.
func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

// Invoke implements tools.Tool.
func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) (any, error) {
	conn := t.Connection
	paramsMap := params.AsMap()
	newStatement, err := tools.ResolveTemplateParams(t.TemplateParameters, t.Statement, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract template params %w", err)
	}

	preparedStatement, err := conn.Prepare(newStatement)
	if err != nil {
		return nil, fmt.Errorf("unable to generate prepared statement %w", err)
	}
	newParamMap, err := getParams(t.Parameters, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract standard params %w", err)
	}

	result, err := conn.Execute(preparedStatement, newParamMap)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer result.Close()
	cols := result.GetColumnNames()
	var out []any
	for result.HasNext() {
		tuple, err := result.Next()
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}
		defer tuple.Close()

		// The result is a tuple, which can be converted to a slice.
		slice, err := tuple.GetAsSlice()
		if err != nil {
			return nil, fmt.Errorf("unable to slice row: %w", err)
		}
		rowMap := make(map[string]interface{})
		for i, col := range cols {
			val := slice[i]
			// Store the value in the map
			rowMap[col] = val
		}
		out = append(out, rowMap)
	}

	return out, nil
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

func getParams(params tools.Parameters, paramValuesMap map[string]interface{}) (map[string]interface{}, error) {
	newParamMap := make(map[string]any)
	for _, p := range params {
		k := p.GetName()
		v, ok := paramValuesMap[k]
		if !ok {
			return nil, fmt.Errorf("missing parameter %s", k)
		}
		newParamMap[k] = v
	}
	return newParamMap, nil
}
