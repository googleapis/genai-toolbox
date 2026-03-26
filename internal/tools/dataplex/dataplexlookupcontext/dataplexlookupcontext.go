// Copyright 2026 Google LLC
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

package dataplexlookupcontext

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/dataplex/apiv1/dataplexpb"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const resourceType string = "dataplex-lookup-context"

func init() {
	if !tools.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", resourceType))
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
	LookupContext(context.Context, string, []string) (*dataplexpb.LookupContextResponse, error)
}

type Config struct {
	Name         string                `yaml:"name" validate:"required"`
	Type         string                `yaml:"type" validate:"required"`
	Source       string                `yaml:"source" validate:"required"`
	Description  string                `yaml:"description"`
	AuthRequired []string              `yaml:"authRequired"`
	Parameters   parameters.Parameters `yaml:"parameters"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	resources := parameters.NewArrayParameter("resources",
		"Required. A list of up to 10 resource names from same project and location.",
		parameters.NewStringParameter("resource",
			"Name of a resource in the following format: projects/{project_id_or_number}/locations/{location}/entryGroups/{group}/entries/{entry}."+
				" Example for a BigQuery table: 'projects/{project_id_or_number}/locations/{location}/entryGroups/@bigquery/entries/bigquery.googleapis.com/projects/{project_id}/datasets/{dataset_id}/tables/{table_id}'."+
				" This is the same value which is returned by the search_entries tool's response in the dataplexEntry.name field."))
	params := parameters.Parameters{resources}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, nil)

	t := Tool{
		Config:     cfg,
		Parameters: params,
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
	Parameters  parameters.Parameters
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	paramsMap := params.AsMap()
	resourcesSlice, err := parameters.ConvertAnySliceToTyped(paramsMap["resources"].([]any), "string")
	if err != nil {
		return nil, util.NewAgentError(fmt.Sprintf("can't convert resources to array of strings: %s", err), err)
	}
	resources := resourcesSlice.([]string)

	if len(resources) == 0 {
		err := fmt.Errorf("resources cannot be empty")
		return nil, util.NewAgentError(err.Error(), err)
	}
	var name string
	for i, resource := range resources {
		parts := strings.Split(resource, "/")
		if len(parts) < 4 || parts[0] != "projects" || parts[2] != "locations" {
			err := fmt.Errorf("invalid resource format at index %d, must be in the format of projects/{project_id_or_number}/locations/{location}/entryGroups/{group}/entries/{entry}", i)
			return nil, util.NewAgentError(err.Error(), err)
		}

		currentName := strings.Join(parts[:4], "/")
		if i == 0 {
			name = currentName
		} else if name != currentName {
			err := fmt.Errorf("all resources must belong to the same project and location. Please make separate calls for each distinct project and location combination")
			return nil, util.NewAgentError(err.Error(), err)
		}
	}

	resp, err := source.LookupContext(ctx, name, resources)
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}
	return resp, nil
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.Parameters, paramValues, embeddingModelsMap, nil)
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

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	return false, nil
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}

func (t Tool) GetParameters() parameters.Parameters {
	return t.Parameters
}
