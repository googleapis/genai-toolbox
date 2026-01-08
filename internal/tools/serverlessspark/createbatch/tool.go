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

package createbatch

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	dataprocpb "cloud.google.com/go/dataproc/v2/apiv1/dataprocpb"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/serverlessspark/common"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type BatchBuilder interface {
	Parameters() parameters.Parameters
	BuildBatch(params parameters.ParamValues) (*dataprocpb.Batch, error)
}

func NewTool(cfg Config, originalCfg tools.ToolConfig, srcs map[string]sources.Source, builder BatchBuilder) (*Tool, error) {
	desc := cfg.Description
	if desc == "" {
		desc = fmt.Sprintf("Creates a Serverless Spark (aka Dataproc Serverless) %s operation.", cfg.Kind)
	}

	allParameters := builder.Parameters()
	inputSchema, _ := allParameters.McpManifest()

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: desc,
		InputSchema: inputSchema,
	}

	return &Tool{
		Config:         cfg,
		originalConfig: originalCfg,
		Builder:        builder,
		manifest:       tools.Manifest{Description: desc, Parameters: allParameters.Manifest()},
		mcpManifest:    mcpManifest,
		Parameters:     allParameters,
	}, nil
}

type Tool struct {
	Config
	originalConfig tools.ToolConfig
	Builder        BatchBuilder
	manifest       tools.Manifest
	mcpManifest    tools.McpManifest
	Parameters     parameters.Parameters
}

func (t *Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Kind)
	if err != nil {
		return nil, err
	}
	client := source.GetBatchControllerClient()

	batch, err := t.Builder.BuildBatch(params)
	if err != nil {
		return nil, fmt.Errorf("failed to build batch: %w", err)
	}

	if t.RuntimeConfig != nil {
		batch.RuntimeConfig = proto.Clone(t.RuntimeConfig).(*dataprocpb.RuntimeConfig)
	}

	if t.EnvironmentConfig != nil {
		batch.EnvironmentConfig = proto.Clone(t.EnvironmentConfig).(*dataprocpb.EnvironmentConfig)
	}

	// Common override for version if present in params
	paramMap := params.AsMap()
	if version, ok := paramMap["version"].(string); ok && version != "" {
		if batch.RuntimeConfig == nil {
			batch.RuntimeConfig = &dataprocpb.RuntimeConfig{}
		}
		batch.RuntimeConfig.Version = version
	}

	req := &dataprocpb.CreateBatchRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", source.GetProject(), source.GetLocation()),
		Batch:  batch,
	}

	op, err := client.CreateBatch(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create batch: %w", err)
	}

	meta, err := op.Metadata()
	if err != nil {
		return nil, fmt.Errorf("failed to get create batch op metadata: %w", err)
	}

	jsonBytes, err := protojson.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create batch op metadata to JSON: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create batch op metadata JSON: %w", err)
	}

	projectID, location, batchID, err := common.ExtractBatchDetails(meta.GetBatch())
	if err != nil {
		return nil, fmt.Errorf("error extracting batch details from name %q: %v", meta.GetBatch(), err)
	}
	consoleUrl := common.BatchConsoleURL(projectID, location, batchID)
	logsUrl := common.BatchLogsURL(projectID, location, batchID, meta.GetCreateTime().AsTime(), time.Time{})

	wrappedResult := map[string]any{
		"opMetadata": meta,
		"consoleUrl": consoleUrl,
		"logsUrl":    logsUrl,
	}

	return wrappedResult, nil
}

func (t *Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Parameters, data, claims)
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.Parameters, paramValues, embeddingModelsMap, nil)
}

func (t *Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t *Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t *Tool) Authorized(services []string) bool {
	return tools.IsAuthorized(t.AuthRequired, services)
}

func (t *Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	return false, nil
}

func (t *Tool) ToConfig() tools.ToolConfig {
	return t.originalConfig
}

func (t *Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}
