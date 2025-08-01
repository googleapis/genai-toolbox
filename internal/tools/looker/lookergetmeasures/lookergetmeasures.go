// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package lookergetmeasures

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	lookersrc "github.com/googleapis/genai-toolbox/internal/sources/looker"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/looker/lookercommon"
	"github.com/googleapis/genai-toolbox/internal/util"

	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const kind string = "looker-get-measures"

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
	Name        string `yaml:"name" validate:"required"`
	Kind        string `yaml:"kind" validate:"required"`
	Source      string `yaml:"source" validate:"required"`
	Description string `yaml:"description" validate:"required"`
	// AuthRequired specifies the authentication services required for this tool.
	// Currently, this field is not actively used for authorization checks within the tool itself,
	// as the Authorized method always returns true. It is included for potential future extensibility.
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
	s, ok := rawS.(*lookersrc.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `looker`", kind)
	}

	modelParameter := tools.NewStringParameter("model", "The model containing the explore.")
	exploreParameter := tools.NewStringParameter("explore", "The explore containing the measures.")
	parameters := tools.Parameters{modelParameter, exploreParameter}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	// finish tool setup
	return Tool{
		Name:         cfg.Name,
		Kind:         kind,
		Parameters:   parameters,
		AuthRequired: cfg.AuthRequired,
		Client:       s.Client,
		ApiSettings:  s.ApiSettings,
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   parameters.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string `yaml:"name"`
	Kind         string `yaml:"kind"`
	Client       *v4.LookerSDK
	ApiSettings  *rtl.ApiSettings
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`
	manifest     tools.Manifest
	mcpManifest  tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) (any, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}
	mapParams := params.AsMap()
	model, ok := mapParams["model"].(string)
	if !ok {
		return nil, fmt.Errorf("'model' must be a string, got %T", mapParams["model"])
	}
	explore, ok := mapParams["explore"].(string)
	if !ok {
		return nil, fmt.Errorf("'explore' must be a string, got %T", mapParams["explore"])
	}

	fields := lookercommon.MeasuresFields
	req := v4.RequestLookmlModelExplore{
		LookmlModelName: model,
		ExploreName:     explore,
		Fields:          &fields,
	}
	resp, err := t.Client.LookmlModelExplore(req, t.ApiSettings)
	if err != nil {
		return nil, fmt.Errorf("error making get_measures request: %w", err)
	}

	if err := lookercommon.CheckLookerExploreFields(&resp); err != nil {
		return nil, fmt.Errorf("error processing get_measures response: %w", err)
	}

	data, err := lookercommon.ExtractLookerFieldProperties(ctx, resp.Fields.Measures)
	if err != nil {
		return nil, fmt.Errorf("error extracting get_measures response: %w", err)
	}
	logger.DebugContext(ctx, "data = ", data)

	return data, nil
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
	// Currently, all Looker tools are considered authorized if the source is correctly configured.
	// The AuthRequired field in the Config struct is reserved for future, more granular authorization.
	return true
}
