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

package fhirpatienteverything

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/common"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/api/googleapi"
)

const resourceType string = "cloud-healthcare-fhir-patient-everything"
const (
	patientIDKey   = "patientID"
	typeFilterKey  = "resourceTypesFilter"
	sinceFilterKey = "sinceFilter"
)

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
	AllowedFHIRStores() map[string]struct{}
	UseClientAuthorization() bool
	FHIRPatientEverything(string, string, string, []googleapi.CallOption) (any, error)
}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Type         string   `yaml:"type" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
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
		return nil, fmt.Errorf("invalid source for %q tool: source %q not compatible", resourceType, cfg.Source)
	}

	idParameter := parameters.NewStringParameter(patientIDKey, "The ID of the patient FHIR resource for which the information is required")
	typeFilterParameter := parameters.NewArrayParameterWithDefault(typeFilterKey, []any{}, "List of FHIR resource types. If provided, only resources of the specified resource type(s) are returned.", parameters.NewStringParameter("resourceType", "A FHIR resource type"))
	sinceFilterParameter := parameters.NewStringParameterWithDefault(sinceFilterKey, "", "If provided, only resources updated after this time are returned. The time uses the format YYYY-MM-DDThh:mm:ss.sss+zz:zz. The time must be specified to the second and include a time zone. For example, 2015-02-07T13:28:17.239+02:00 or 2017-01-01T00:00:00Z")
	params := parameters.Parameters{idParameter, typeFilterParameter, sinceFilterParameter}
	if len(s.AllowedFHIRStores()) != 1 {
		params = append(params, parameters.NewStringParameter(common.StoreKey, "The FHIR store ID to retrieve the resource from."))
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, nil)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		Parameters:  params,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	Parameters  parameters.Parameters `yaml:"parameters"`
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

	storeID, err := common.ValidateAndFetchStoreID(params, source.AllowedFHIRStores())
	if err != nil {
		// ValidateAndFetchStoreID usually returns input validation errors
		return nil, util.NewAgentError("failed to validate store ID", err)
	}
	patientID, ok := params.AsMap()[patientIDKey].(string)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("invalid or missing '%s' parameter; expected a string", patientIDKey), nil)
	}

	var tokenStr string
	if source.UseClientAuthorization() {
		tokenStr, err = accessToken.ParseBearerToken()
		if err != nil {
			return nil, util.NewClientServerError("error parsing access token", http.StatusUnauthorized, err)
		}
	}

	var opts []googleapi.CallOption
	if val, ok := params.AsMap()[typeFilterKey]; ok {
		types, ok := val.([]any)
		if !ok {
			return nil, util.NewAgentError(fmt.Sprintf("invalid '%s' parameter; expected a string array", typeFilterKey), nil)
		}
		typeFilterSlice, err := parameters.ConvertAnySliceToTyped(types, "string")
		if err != nil {
			return nil, util.NewAgentError(fmt.Sprintf("can't convert '%s' to array of strings: %s", typeFilterKey, err), err)
		}
		if len(typeFilterSlice.([]string)) != 0 {
			opts = append(opts, googleapi.QueryParameter("_type", strings.Join(typeFilterSlice.([]string), ",")))
		}
	}
	if since, ok := params.AsMap()[sinceFilterKey]; ok {
		sinceStr, ok := since.(string)
		if !ok {
			return nil, util.NewAgentError(fmt.Sprintf("invalid '%s' parameter; expected a string", sinceFilterKey), nil)
		}
		if sinceStr != "" {
			opts = append(opts, googleapi.QueryParameter("_since", sinceStr))
		}
	}

	resp, err := source.FHIRPatientEverything(storeID, patientID, tokenStr, opts)
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}
	return resp, nil
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.Parameters, paramValues, embeddingModelsMap, nil)
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

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}

func (t Tool) GetParameters() parameters.Parameters {
	return t.Parameters
}
