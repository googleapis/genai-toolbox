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
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	healthcareds "github.com/googleapis/genai-toolbox/internal/sources/cloudhealthcare"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/common"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/healthcare/v1"
)

const kind string = "cloud-healthcare-fhir-patient-everything"
const (
	patientIDKey   = "patientID"
	typeFilterKey  = "resourceTypesFilter"
	sinceFilterKey = "sinceFilter"
)

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
	Project() string
	Region() string
	DatasetID() string
	AllowedFHIRStores() map[string]struct{}
	Service() *healthcare.Service
	ServiceCreator() healthcareds.HealthcareServiceCreator
	UseClientAuthorization() bool
}

// validate compatible sources are still compatible
var _ compatibleSource = &healthcareds.Source{}

var compatibleSources = [...]string{healthcareds.SourceKind}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
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
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
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
		Config:         cfg,
		Parameters:     params,
		Project:        s.Project(),
		Region:         s.Region(),
		Dataset:        s.DatasetID(),
		AllowedStores:  s.AllowedFHIRStores(),
		UseClientOAuth: s.UseClientAuthorization(),
		ServiceCreator: s.ServiceCreator(),
		Service:        s.Service(),
		manifest:       tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:    mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	UseClientOAuth bool                  `yaml:"useClientOAuth"`
	Parameters     parameters.Parameters `yaml:"parameters"`

	Project, Region, Dataset string
	AllowedStores            map[string]struct{}
	Service                  *healthcare.Service
	ServiceCreator           healthcareds.HealthcareServiceCreator
	manifest                 tools.Manifest
	mcpManifest              tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	storeID, err := common.ValidateAndFetchStoreID(params, t.AllowedStores)
	if err != nil {
		return nil, err
	}
	patientID, ok := params.AsMap()[patientIDKey].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing '%s' parameter; expected a string", patientIDKey)
	}

	svc := t.Service
	// Initialize new service if using user OAuth token
	if t.UseClientOAuth {
		tokenStr, err := accessToken.ParseBearerToken()
		if err != nil {
			return nil, fmt.Errorf("error parsing access token: %w", err)
		}
		svc, err = t.ServiceCreator(tokenStr)
		if err != nil {
			return nil, fmt.Errorf("error creating service from OAuth access token: %w", err)
		}
	}

	name := fmt.Sprintf("projects/%s/locations/%s/datasets/%s/fhirStores/%s/fhir/Patient/%s", t.Project, t.Region, t.Dataset, storeID, patientID)
	var opts []googleapi.CallOption
	if val, ok := params.AsMap()[typeFilterKey]; ok {
		types, ok := val.([]any)
		if !ok {
			return nil, fmt.Errorf("invalid '%s' parameter; expected a string array", typeFilterKey)
		}
		typeFilterSlice, err := parameters.ConvertAnySliceToTyped(types, "string")
		if err != nil {
			return nil, fmt.Errorf("can't convert '%s' to array of strings: %s", typeFilterKey, err)
		}
		if len(typeFilterSlice.([]string)) != 0 {
			opts = append(opts, googleapi.QueryParameter("_type", strings.Join(typeFilterSlice.([]string), ",")))
		}
	}
	if since, ok := params.AsMap()[sinceFilterKey]; ok {
		sinceStr, ok := since.(string)
		if !ok {
			return nil, fmt.Errorf("invalid '%s' parameter; expected a string", sinceFilterKey)
		}
		if sinceStr != "" {
			opts = append(opts, googleapi.QueryParameter("_since", sinceStr))
		}
	}

	resp, err := svc.Projects.Locations.Datasets.FhirStores.Fhir.PatientEverything(name).Do(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to call patient everything for %q: %w", name, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response: %w", err)
	}
	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("patient-everything: status %d %s: %s", resp.StatusCode, resp.Status, respBytes)
	}
	var jsonMap map[string]interface{}
	if err := json.Unmarshal([]byte(string(respBytes)), &jsonMap); err != nil {
		return nil, fmt.Errorf("could not unmarshal response as json: %w", err)
	}
	return jsonMap, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Parameters, data, claims)
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

func (t Tool) RequiresClientAuthorization() bool {
	return t.UseClientOAuth
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
