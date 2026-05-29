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

package dataplexgeneratedataprofile

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const resourceType string = "dataplex-generate-data-profile"

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
	ProjectID() string
	GenerateDataProfile(ctx context.Context, projectID, location, resourcePath string, publish bool) (string, error)
}

type Config struct {
	Name         string                 `yaml:"name" validate:"required"`
	Type         string                 `yaml:"type" validate:"required"`
	Source       string                 `yaml:"source" validate:"required"`
	Description  string                 `yaml:"description"`
	AuthRequired []string               `yaml:"authRequired"`
	Annotations  *tools.ToolAnnotations `yaml:"annotations,omitempty"`

	ScopesRequired []string `yaml:"scopesRequired"`
}

var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("source %q not found", cfg.Source)
	}
	_, ok = rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source %q not compatible", resourceType, cfg.Source)
	}

	resourcePath := parameters.NewStringParameter("resourcePath", "Required. The BigQuery table to analyze. Accepts raw table name (e.g. 'my_table'), dataset.table (e.g. 'my_dataset.my_table'), or fully-qualified resource path (e.g. '//bigquery.googleapis.com/projects/{project}/datasets/{dataset}/tables/{table}').")
	location := parameters.NewStringParameter("location", "Required. The Google Cloud region where the Dataplex scan should be created and executed (e.g., 'us-central1'). This should match the location of the BigQuery resource.")
	publish := parameters.NewBooleanParameter("publish", "Required. Whether to publish the generated profile results to the Dataplex Universal Catalog.")

	params := parameters.Parameters{resourcePath, location, publish}

	t := Tool{
		Config:     cfg,
		Parameters: params,
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   params.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
	}
	return t, nil
}

type Tool struct {
	Config
	Parameters parameters.Parameters
	manifest   tools.Manifest
}

func (t Tool) GetName() string {
	return t.Name
}

func (t Tool) GetDescription() string {
	return t.Description
}

func (t Tool) GetAuthRequired() []string {
	return t.AuthRequired
}

func (t Tool) GetAnnotations() *tools.ToolAnnotations {
	return tools.GetAnnotationsOrDefault(t.Annotations, tools.NewDestructiveAnnotations)
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
	resourcePath, _ := paramsMap["resourcePath"].(string)
	location, _ := paramsMap["location"].(string)
	publish, _ := paramsMap["publish"].(bool)

	if resourcePath == "" {
		return nil, util.NewAgentError("resourcePath parameter is required", nil)
	}
	if location == "" {
		return nil, util.NewAgentError("location parameter is required", nil)
	}

	projectId := source.ProjectID()

	// Smart BigQuery path normalization to prevent ResourceName errors in Dataplex
	if !strings.HasPrefix(resourcePath, "//bigquery.googleapis.com/") {
		if strings.HasPrefix(resourcePath, "projects/") {
			resourcePath = "//bigquery.googleapis.com/" + resourcePath
		} else {
			parts := strings.Split(resourcePath, ".")
			if len(parts) == 3 {
				resourcePath = fmt.Sprintf("//bigquery.googleapis.com/projects/%s/datasets/%s/tables/%s", parts[0], parts[1], parts[2])
			} else if len(parts) == 2 {
				resourcePath = fmt.Sprintf("//bigquery.googleapis.com/projects/%s/datasets/%s/tables/%s", projectId, parts[0], parts[1])
			}
		}
	}

	opName, err := source.GenerateDataProfile(ctx, projectId, location, resourcePath, publish)
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}

	return map[string]string{
		"operation_id": opName,
	}, nil
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.Parameters, paramValues, embeddingModelsMap, nil)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
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

func (t Tool) GetScopesRequired() []string {
	return t.ScopesRequired
}
