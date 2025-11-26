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

package dataplexlookupentry

import (
	"context"
	"fmt"

	dataplexapi "cloud.google.com/go/dataplex/apiv1"
	dataplexpb "cloud.google.com/go/dataplex/apiv1/dataplexpb"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	dataplexds "github.com/googleapis/genai-toolbox/internal/sources/dataplex"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind string = "dataplex-lookup-entry"

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
	CatalogClient() *dataplexapi.CatalogClient
}

// validate compatible sources are still compatible
var _ compatibleSource = &dataplexds.Source{}

var compatibleSources = [...]string{dataplexds.SourceKind}

type Config struct {
	Name         string                `yaml:"name" validate:"required"`
	Kind         string                `yaml:"kind" validate:"required"`
	Source       string                `yaml:"source" validate:"required"`
	Description  string                `yaml:"description"`
	AuthRequired []string              `yaml:"authRequired"`
	Parameters   parameters.Parameters `yaml:"parameters"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// Initialize the search configuration with the provided sources
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}
	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	viewDesc := `
				## Argument: view

				**Type:** Integer

				**Description:** Specifies the parts of the entry and its aspects to return.

				**Possible Values:**

				*   1 (BASIC): Returns entry without aspects.
				*   2 (FULL): Return all required aspects and the keys of non-required aspects. (Default)
				*   3 (CUSTOM): Return the entry and aspects requested in aspect_types field (at most 100 aspects). Always use this view when aspect_types is not empty.
				*   4 (ALL): Return the entry and both required and optional aspects (at most 100 aspects)
				`

	name := parameters.NewStringParameter("name", "The project to which the request should be attributed in the following form: projects/{project}/locations/{location}.")
	view := parameters.NewIntParameterWithDefault("view", 2, viewDesc)
	aspectTypes := parameters.NewArrayParameterWithDefault("aspectTypes", []any{}, "Limits the aspects returned to the provided aspect types. It only works when used together with CUSTOM view.", parameters.NewStringParameter("aspectType", "The types of aspects to be included in the response in the format `projects/{project}/locations/{location}/aspectTypes/{aspectType}`."))
	entry := parameters.NewStringParameter("entry", "The resource name of the Entry in the following form: projects/{project}/locations/{location}/entryGroups/{entryGroup}/entries/{entry}.")
	params := parameters.Parameters{name, view, aspectTypes, entry}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, nil)

	t := Tool{
		Config:        cfg,
		Parameters:    params,
		CatalogClient: s.CatalogClient(),
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   params.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

type Tool struct {
	Config
	Parameters    parameters.Parameters
	CatalogClient *dataplexapi.CatalogClient
	manifest      tools.Manifest
	mcpManifest   tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()
	viewMap := map[int]dataplexpb.EntryView{
		1: dataplexpb.EntryView_BASIC,
		2: dataplexpb.EntryView_FULL,
		3: dataplexpb.EntryView_CUSTOM,
		4: dataplexpb.EntryView_ALL,
	}
	name, _ := paramsMap["name"].(string)
	entry, _ := paramsMap["entry"].(string)
	view, _ := paramsMap["view"].(int)
	aspectTypeSlice, err := parameters.ConvertAnySliceToTyped(paramsMap["aspectTypes"].([]any), "string")
	if err != nil {
		return nil, fmt.Errorf("can't convert aspectTypes to array of strings: %s", err)
	}
	aspectTypes := aspectTypeSlice.([]string)

	req := &dataplexpb.LookupEntryRequest{
		Name:        name,
		View:        viewMap[view],
		AspectTypes: aspectTypes,
		Entry:       entry,
	}

	result, err := t.CatalogClient.LookupEntry(ctx, req)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	// Parse parameters from the provided data
	return parameters.ParseParams(t.Parameters, data, claims)
}

func (t Tool) Manifest() tools.Manifest {
	// Returns the tool manifest
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	// Returns the tool MCP manifest
	return t.mcpManifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) RequiresClientAuthorization() bool {
	return false
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
