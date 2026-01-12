// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package serverlesssparklistbatches

import (
	"context"
	"fmt"

	dataproc "cloud.google.com/go/dataproc/v2/apiv1"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind = "serverless-spark-list-batches"

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
	GetBatchControllerClient() *dataproc.BatchControllerClient
	ListBatches(context.Context, *int, string, string) (any, error)
}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

// ToolConfigKind returns the unique name for this tool.
func (cfg Config) ToolConfigKind() string {
	return kind
}

// Initialize creates a new Tool instance.
func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	desc := cfg.Description
	if desc == "" {
		desc = "Lists available Serverless Spark (aka Dataproc Serverless) batches"
	}

	allParameters := parameters.Parameters{
		parameters.NewStringParameterWithRequired("filter", `Filter expression to limit the batches. Filters are case sensitive, and may contain multiple clauses combined with logical operators (AND/OR, case sensitive). Supported fields are batch_id, batch_uuid, state, create_time, and labels. e.g. state = RUNNING AND create_time < "2023-01-01T00:00:00Z" filters for batches in state RUNNING that were created before 2023-01-01. state = RUNNING AND labels.environment=production filters for batches in state in a RUNNING state that have a production environment label. Valid states are STATE_UNSPECIFIED, PENDING, RUNNING, CANCELLING, CANCELLED, SUCCEEDED, FAILED. Valid operators are < > <= >= = !=, and : as "has" for labels, meaning any non-empty value)`, false),
		parameters.NewIntParameterWithDefault("pageSize", 20, "The maximum number of batches to return in a single page (default 20)"),
		parameters.NewStringParameterWithRequired("pageToken", "A page token, received from a previous `ListBatches` call", false),
	}
	inputSchema, _ := allParameters.McpManifest()

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: desc,
		InputSchema: inputSchema,
	}

	return Tool{
		Config:      cfg,
		manifest:    tools.Manifest{Description: desc, Parameters: allParameters.Manifest()},
		mcpManifest: mcpManifest,
		Parameters:  allParameters,
	}, nil
}

// Tool is the implementation of the tool.
type Tool struct {
	Config
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
	Parameters  parameters.Parameters
}

// Invoke executes the tool's operation.
func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Kind)
	if err != nil {
		return nil, err
	}
	paramMap := params.AsMap()
	var pageSize *int
	if ps, ok := paramMap["pageSize"]; ok && ps != nil {
		pageSizeV := ps.(int)
		if pageSizeV <= 0 {
			return nil, fmt.Errorf("pageSize must be positive: %d", pageSizeV)
		}
		pageSize = &pageSizeV
	}
	pt, _ := paramMap["pageToken"].(string)
	filter, _ := paramMap["filter"].(string)
	return source.ListBatches(ctx, pageSize, pt, filter)
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Parameters, data, claims)
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

func (t Tool) Authorized(services []string) bool {
	return tools.IsAuthorized(t.AuthRequired, services)
}

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	// Client OAuth not supported, rely on ADCs.
	return false, nil
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}
