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

package bigquerylisttableids

import (
	"context"
	"fmt"
	"sort"
	"strings"

	bigqueryapi "cloud.google.com/go/bigquery"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	bigqueryds "github.com/googleapis/genai-toolbox/internal/sources/bigquery"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"google.golang.org/api/iterator"
)

const kind string = "bigquery-list-table-ids"
const projectKey string = "project"
const datasetKey string = "dataset"

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
	BigQueryClient() *bigqueryapi.Client
	BigQueryClientCreator() bigqueryds.BigqueryClientCreator
	BigQueryProject() string
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
	projectDescription := "The Google Cloud project ID containing the dataset."
	datasetDescription := "The dataset to list table ids."
	var datasetParameter tools.Parameter
	allowedDatasets := s.BigQueryAllowedDatasets()
	if len(allowedDatasets) > 0 {
		if len(allowedDatasets) == 1 {
			parts := strings.Split(allowedDatasets[0], ".")
			defaultProjectID = parts[0]
			datasetID := parts[1]
			projectDescription += fmt.Sprintf(" Must be `%s`.", defaultProjectID)
			datasetDescription += fmt.Sprintf(" Must be `%s`.", datasetID)
			datasetParameter = tools.NewStringParameterWithDefault(datasetKey, datasetID, datasetDescription)
		} else {
			datasetIDsByProject := make(map[string][]string)
			for _, ds := range allowedDatasets {
				parts := strings.Split(ds, ".")
				project := parts[0]
				dataset := parts[1]
				datasetIDsByProject[project] = append(datasetIDsByProject[project], fmt.Sprintf("`%s`", dataset))
			}

			var datasetDescriptions, projectIDList []string
			for project, datasets := range datasetIDsByProject {
				sort.Strings(datasets)
				datasetList := strings.Join(datasets, ", ")
				datasetDescriptions = append(datasetDescriptions, fmt.Sprintf("%s from project `%s`", datasetList, project))
			}

			var projectIDList []string
			for p := range projectIDs {
				projectIDList = append(projectIDList, fmt.Sprintf("`%s`", p))
			}
			projectDescription += fmt.Sprintf(" Must be one of the following: %s.", strings.Join(projectIDList, ", "))
			datasetDescription += fmt.Sprintf(" Must be one of the allowed datasets: %s.", strings.Join(datasetDescriptions, ", "))
			datasetParameter = tools.NewStringParameter(datasetKey, datasetDescription)
		}
	} else {
		datasetParameter = tools.NewStringParameter(datasetKey, datasetDescription)
	}
	projectParameter := tools.NewStringParameterWithDefault(projectKey, defaultProjectID, projectDescription)

	parameters := tools.Parameters{projectParameter, datasetParameter}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	// finish tool setup
	t := Tool{
		Name:             cfg.Name,
		Kind:             kind,
		Parameters:       parameters,
		AuthRequired:     cfg.AuthRequired,
		UseClientOAuth:   s.UseClientAuthorization(),
		ClientCreator:    s.BigQueryClientCreator(),
		Client:           s.BigQueryClient(),
		IsDatasetAllowed: s.IsDatasetAllowed,
		manifest:         tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:      mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name           string           `yaml:"name"`
	Kind           string           `yaml:"kind"`
	AuthRequired   []string         `yaml:"authRequired"`
	UseClientOAuth bool             `yaml:"useClientOAuth"`
	Parameters     tools.Parameters `yaml:"parameters"`

	Client           *bigqueryapi.Client
	ClientCreator    bigqueryds.BigqueryClientCreator
	IsDatasetAllowed func(projectID, datasetID string) bool
	Statement        string
	manifest         tools.Manifest
	mcpManifest      tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	mapParams := params.AsMap()
	projectId, ok := mapParams[projectKey].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing '%s' parameter; expected a string", projectKey)
	}

	datasetId, ok := mapParams[datasetKey].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing '%s' parameter; expected a string", datasetKey)
	}

	if !t.IsDatasetAllowed(projectId, datasetId) {
		return nil, fmt.Errorf("access denied to dataset '%s' because it is not in the configured list of allowed datasets for project '%s'", datasetId, projectId)
	}

	if !t.IsDatasetAllowed(projectId, datasetId) {
		return nil, fmt.Errorf("access denied to dataset '%s' because it is not in the configured list of allowed datasets for project '%s'", datasetId, projectId)
	}

	bqClient := t.Client
	var err error
	// Initialize new client if using user OAuth token
	if t.UseClientOAuth {
		bqClient, _, err = t.ClientCreator(accessToken, false)
		if err != nil {
			return nil, fmt.Errorf("error creating client from OAuth access token: %w", err)
		}
	}

	dsHandle := bqClient.DatasetInProject(projectId, datasetId)

	var tableIds []any
	tableIterator := dsHandle.Tables(ctx)
	for {
		table, err := tableIterator.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate through tables in dataset %s.%s: %w", projectId, datasetId, err)
		}

		// Remove leading and trailing quotes
		id := table.TableID
		if len(id) >= 2 && id[0] == '"' && id[len(id)-1] == '"' {
			id = id[1 : len(id)-1]
		}
		tableIds = append(tableIds, id)
	}

	return tableIds, nil
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
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) RequiresClientAuthorization() bool {
	return t.UseClientOAuth
}
