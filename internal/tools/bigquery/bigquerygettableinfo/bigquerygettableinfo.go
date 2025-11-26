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

package bigquerygettableinfo

import (
	"context"
	"fmt"

	bigqueryapi "cloud.google.com/go/bigquery"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	bigqueryds "github.com/googleapis/genai-toolbox/internal/sources/bigquery"
	"github.com/googleapis/genai-toolbox/internal/tools"
	bqutil "github.com/googleapis/genai-toolbox/internal/tools/bigquery/bigquerycommon"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind string = "bigquery-get-table-info"
const projectKey string = "project"
const datasetKey string = "dataset"
const tableKey string = "table"

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
	BigQueryProject() string
	BigQueryClient() *bigqueryapi.Client
	BigQueryClientCreator() bigqueryds.BigqueryClientCreator
	UseClientAuthorization() bool
	IsDatasetAllowed(projectID, datasetID string) bool
	BigQueryAllowedDatasets() []string
}

// validate compatible sources are still compatible
var _ compatibleSource = &bigqueryds.Source{}

var compatibleSources = [...]string{bigqueryds.SourceKind}

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

	defaultProjectID := s.BigQueryProject()
	projectDescription := "The Google Cloud project ID containing the dataset and table."
	datasetDescription := "The table's parent dataset."
	var datasetParameter parameters.Parameter
	var projectParameter parameters.Parameter

	projectParameter, datasetParameter = bqutil.InitializeDatasetParameters(
		s.BigQueryAllowedDatasets(),
		defaultProjectID,
		projectKey, datasetKey,
		projectDescription, datasetDescription,
	)

	tableParameter := parameters.NewStringParameter(tableKey, "The table to get metadata information.")
	params := parameters.Parameters{projectParameter, datasetParameter, tableParameter}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, nil)

	// finish tool setup
	t := Tool{
		Config:           cfg,
		Parameters:       params,
		UseClientOAuth:   s.UseClientAuthorization(),
		ClientCreator:    s.BigQueryClientCreator(),
		Client:           s.BigQueryClient(),
		IsDatasetAllowed: s.IsDatasetAllowed,
		manifest:         tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:      mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	UseClientOAuth bool                  `yaml:"useClientOAuth"`
	Parameters     parameters.Parameters `yaml:"parameters"`

	Client           *bigqueryapi.Client
	ClientCreator    bigqueryds.BigqueryClientCreator
	Statement        string
	IsDatasetAllowed func(projectID, datasetID string) bool
	manifest         tools.Manifest
	mcpManifest      tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	mapParams := params.AsMap()
	projectId, ok := mapParams[projectKey].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing '%s' parameter; expected a string", projectKey)
	}

	datasetId, ok := mapParams[datasetKey].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing '%s' parameter; expected a string", datasetKey)
	}

	tableId, ok := mapParams[tableKey].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing '%s' parameter; expected a string", tableKey)
	}

	if !t.IsDatasetAllowed(projectId, datasetId) {
		return nil, fmt.Errorf("access denied to dataset '%s' because it is not in the configured list of allowed datasets for project '%s'", datasetId, projectId)
	}

	bqClient := t.Client

	var err error
	// Initialize new client if using user OAuth token
	if t.UseClientOAuth {
		tokenStr, err := accessToken.ParseBearerToken()
		if err != nil {
			return nil, fmt.Errorf("error parsing access token: %w", err)
		}
		bqClient, _, err = t.ClientCreator(tokenStr, false)
		if err != nil {
			return nil, fmt.Errorf("error creating client from OAuth access token: %w", err)
		}
	}

	dsHandle := bqClient.DatasetInProject(projectId, datasetId)
	tableHandle := dsHandle.Table(tableId)

	metadata, err := tableHandle.Metadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata for table %s.%s.%s: %w", projectId, datasetId, tableId, err)
	}

	return metadata, nil
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
