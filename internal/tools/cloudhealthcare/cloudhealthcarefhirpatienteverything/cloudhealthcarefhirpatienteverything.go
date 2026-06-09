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
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/tools/cloudhealthcare/common"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
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
	actual := Config{ConfigBase: tools.ConfigBase{Name: name}}
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
	tools.ConfigBase `yaml:",inline"`
	Type             string                 `yaml:"type" validate:"required"`
	Source           string                 `yaml:"source" validate:"required"`
	Annotations      *tools.ToolAnnotations `yaml:"annotations,omitempty"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	if cfg.Description == "" {
		return nil, fmt.Errorf("description is required for tool %q", cfg.Name)
	}

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
	allParameters := parameters.Parameters{idParameter, typeFilterParameter, sinceFilterParameter}
	if len(s.AllowedFHIRStores()) != 1 {
		allParameters = append(allParameters, parameters.NewStringParameter(common.StoreKey, "The FHIR store ID to retrieve the resource from."))
	}

	return Tool{
		BaseTool: tools.NewBaseTool(
			cfg,
			tools.GetAnnotationsOrDefault(cfg.Annotations, tools.NewReadOnlyAnnotations),
			tools.Manifest{Description: cfg.Description, Parameters: allParameters.Manifest(), AuthRequired: cfg.AuthRequired},
			allParameters,
		),
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	tools.BaseTool[Config]
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Cfg
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
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

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}
