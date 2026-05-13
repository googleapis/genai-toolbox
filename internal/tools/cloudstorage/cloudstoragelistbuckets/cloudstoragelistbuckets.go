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

package cloudstoragelistbuckets

import (
	"context"
	"fmt"
	"net/http"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/tools/cloudstorage/cloudstoragecommon"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const resourceType string = "cloud-storage-list-buckets"

// maxResultsLimit matches the GCS per-page cap. Values above this are rejected
// in Invoke so callers see an explicit error instead of a silently-clamped page.
const maxResultsLimit = 1000

const (
	projectKey    = "project"
	prefixKey     = "prefix"
	maxResultsKey = "max_results"
	pageTokenKey  = "page_token"
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
	ListBuckets(ctx context.Context, project, prefix string, maxResults int, pageToken string) (map[string]any, error)
}

type Config struct {
	Name         string                 `yaml:"name" validate:"required"`
	Type         string                 `yaml:"type" validate:"required"`
	Source       string                 `yaml:"source" validate:"required"`
	Description  string                 `yaml:"description" validate:"required"`
	AuthRequired []string               `yaml:"authRequired"`
	Annotations  *tools.ToolAnnotations `yaml:"annotations,omitempty"`

	ScopesRequired []string `yaml:"scopesRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	projectParam := parameters.NewStringParameterWithDefault(projectKey, "", "Project ID to list buckets in. When empty, the source's configured project is used.")
	prefixParam := parameters.NewStringParameterWithDefault(prefixKey, "", "Filter results to buckets whose names begin with this prefix.")
	maxResultsParam := parameters.NewIntParameterWithDefault(maxResultsKey, 0, "Maximum number of buckets to return per page. A value of 0 uses the API default (1000); negative values and values above 1000 are rejected.")
	pageTokenParam := parameters.NewStringParameterWithDefault(pageTokenKey, "", "A previously-returned page token for retrieving the next page of results.")
	params := parameters.Parameters{projectParam, prefixParam, maxResultsParam, pageTokenParam}

	annotations := tools.GetAnnotationsOrDefault(cfg.Annotations, tools.NewReadOnlyAnnotations)
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, annotations)

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

	mapParams := params.AsMap()
	project, _ := mapParams[projectKey].(string)
	prefix, _ := mapParams[prefixKey].(string)
	pageToken, _ := mapParams[pageTokenKey].(string)
	maxResults, _ := mapParams[maxResultsKey].(int)
	if maxResults < 0 {
		return nil, util.NewAgentError(fmt.Sprintf("invalid '%s' parameter: %d must be >= 0 (use 0 for the API default)", maxResultsKey, maxResults), nil)
	}
	if maxResults > maxResultsLimit {
		return nil, util.NewAgentError(fmt.Sprintf("invalid '%s' parameter: %d exceeds the maximum of %d", maxResultsKey, maxResults, maxResultsLimit), nil)
	}

	resp, err := source.ListBuckets(ctx, project, prefix, maxResults, pageToken)
	if err != nil {
		return nil, cloudstoragecommon.ProcessGCSError(err)
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
