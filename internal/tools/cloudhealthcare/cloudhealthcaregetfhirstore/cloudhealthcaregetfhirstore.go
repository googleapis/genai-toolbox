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

package getfhirstore

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	healthcareds "github.com/googleapis/genai-toolbox/internal/sources/cloudhealthcare"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/common"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/api/healthcare/v1"
)

const kind string = "cloud-healthcare-get-fhir-store"

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
		return nil, fmt.Errorf("invalid source for %q tool: source %q not compatible", kind, cfg.Source)
	}

	params := parameters.Parameters{}
	if len(s.AllowedFHIRStores()) != 1 {
		params = append(params, parameters.NewStringParameter(common.StoreKey, "The FHIR store ID to get details for."))
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params)

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

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	s, ok := resourceMgr.GetSource(t.Source)
	if !ok {
		return nil, fmt.Errorf("unable to retrieve source %s in tool %s", t.Source, t.Name)
	}
	source, ok := s.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source %q not compatible", kind, t.Source)
	}

	storeID, err := common.ValidateAndFetchStoreID(params, source.AllowedFHIRStores())
	if err != nil {
		return nil, err
	}

	svc := source.Service()
	// Initialize new service if using user OAuth token
	if source.UseClientAuthorization() {
		tokenStr, err := accessToken.ParseBearerToken()
		if err != nil {
			return nil, fmt.Errorf("error parsing access token: %w", err)
		}
		svc, err = source.ServiceCreator()(tokenStr)
		if err != nil {
			return nil, fmt.Errorf("error creating service from OAuth access token: %w", err)
		}
	}

	storeName := fmt.Sprintf("projects/%s/locations/%s/datasets/%s/fhirStores/%s", source.Project(), source.Region(), source.DatasetID(), storeID)
	store, err := svc.Projects.Locations.Datasets.FhirStores.Get(storeName).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get FHIR store %q: %w", storeName, err)
	}
	return store, nil
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

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	s, ok := resourceMgr.GetSource(t.Source)
	if !ok {
		return false, fmt.Errorf("unable to retrieve source %s in tool %s", t.Source, t.Name)
	}

	source, ok := s.(compatibleSource)
	if !ok {
		return false, fmt.Errorf("invalid source for %q tool: source %q not compatible", kind, t.Source)
	}
	return source.UseClientAuthorization(), nil
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}
