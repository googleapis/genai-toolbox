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
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`
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

	name := tools.NewStringParameter("name", "The project to which the request should be attributed in the following form: projects/{project}/locations/{location}.")
	view := tools.NewStringParameterWithDefault("view", string(dataplexpb.EntryView_FULL), "View to control which parts of an entry the service should return.")
	//aspectTypes := tools.NewArrayParameterWithDefault("aspectTypes", []string{}, "Limits the aspects returned to the provided aspect types. It only works for CUSTOM view.", tools.StringParameter{})
	//paths := tools.NewArrayParameterWithDefault("paths", []string{}, "The paths of the entries to look up. If not specified, the entry will be looked up by name.", tools.StringParameter{})
	entry := tools.NewStringParameter("entry", "The resource name of the Entry in the following form: projects/{project}/locations/{location}/entryGroups/{entryGroup}/entries/{entry}.")
	parameters := tools.Parameters{name, view, entry}

	//_, paramManifest, paramMcpManifest := tools.ProcessParameters(nil, cfg.Parameters)

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	t := &LookupTool{
		Name:          cfg.Name,
		Kind:          kind,
		Parameters:    parameters,
		AuthRequired:  cfg.AuthRequired,
		CatalogClient: s.CatalogClient(),
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   parameters.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

type LookupTool struct {
	Name          string
	Kind          string
	Parameters    tools.Parameters
	AuthRequired  []string
	CatalogClient *dataplexapi.CatalogClient
	manifest      tools.Manifest
	mcpManifest   tools.McpManifest
}

func (t *LookupTool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t *LookupTool) Invoke(ctx context.Context, params tools.ParamValues) (any, error) {
	paramsMap := params.AsMap()
	name, _ := paramsMap["name"].(string)
	entry, _ := paramsMap["entry"].(string)
	view, _ := paramsMap["view"].(dataplexpb.EntryView)
	//aspect_types, _ := paramsMap["aspect_types"].([]string)

	req := &dataplexpb.LookupEntryRequest{
		Name: name,
		View: view,
		//AspectTypes: aspect_types,
		Entry: entry,
	}

	result, err := t.CatalogClient.LookupEntry(ctx, req)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *LookupTool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	// Parse parameters from the provided data
	return tools.ParseParams(t.Parameters, data, claims)
}

func (t *LookupTool) Manifest() tools.Manifest {
	// Returns the tool manifest
	return t.manifest
}

func (t *LookupTool) McpManifest() tools.McpManifest {
	// Returns the tool MCP manifest
	return t.mcpManifest
}
