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

package oceanbasesql

import (
	"context"
	"database/sql"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind string = "oceanbase-sql"

func init() {
	if !tools.Register(kind, newConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", kind))
	}
}

type compatibleSource interface {
	OceanBasePool() *sql.DB
	RunSQL(context.Context, string, []any) (any, error)
}

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

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	allParameters, paramManifest, err := parameters.ProcessParameters(cfg.TemplateParameters, cfg.Parameters)
	if err != nil {
		return nil, fmt.Errorf("unable to process parameters: %w", err)
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		AllParams:   allParameters,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	AllParams   parameters.Parameters `yaml:"allParams"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// Invoke executes the SQL statement with the provided parameters.
func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Kind)
	if err != nil {
		return nil, err
	}

	paramsMap := params.AsMap()
	newStatement, err := parameters.ResolveTemplateParams(t.TemplateParameters, t.Statement, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract template params %w", err)
	}

	newParams, err := parameters.GetParams(t.Parameters, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract standard params %w", err)
	}
	sliceParams := newParams.AsSlice()
	return source.RunSQL(ctx, newStatement, sliceParams)
}

// ParseParams parses the input parameters for the tool.
func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.AllParams, data, claims)
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.AllParams, paramValues, embeddingModelsMap, nil)
}

// Manifest returns the tool manifest.
func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

// McpManifest returns the MCP manifest for the tool.
func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

// Authorized checks if the tool is authorized for the given auth services.
func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	return false, nil
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}
