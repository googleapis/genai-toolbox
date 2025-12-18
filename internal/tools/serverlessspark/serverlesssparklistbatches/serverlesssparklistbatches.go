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
	"time"

	"cloud.google.com/go/dataproc/v2/apiv1/dataprocpb"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/serverlessspark"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/serverlessspark/common"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/api/iterator"
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
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("source %q not found", cfg.Source)
	}

	ds, ok := rawS.(*serverlessspark.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `%s`", kind, serverlessspark.SourceKind)
	}

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
		Source:      ds,
		manifest:    tools.Manifest{Description: desc, Parameters: allParameters.Manifest()},
		mcpManifest: mcpManifest,
		Parameters:  allParameters,
	}, nil
}

// Tool is the implementation of the tool.
type Tool struct {
	Config

	Source *serverlessspark.Source

	manifest    tools.Manifest
	mcpManifest tools.McpManifest
	Parameters  parameters.Parameters
}

// ListBatchesResponse is the response from the list batches API.
type ListBatchesResponse struct {
	Batches       []Batch `json:"batches"`
	NextPageToken string  `json:"nextPageToken"`
}

// Batch represents a single batch job.
type Batch struct {
	Name       string `json:"name"`
	UUID       string `json:"uuid"`
	State      string `json:"state"`
	Creator    string `json:"creator"`
	CreateTime string `json:"createTime"`
	Operation  string `json:"operation"`
	ConsoleURL string `json:"consoleUrl"`
	LogsURL    string `json:"logsUrl"`
}

// Invoke executes the tool's operation.
func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	client := t.Source.GetBatchControllerClient()

	parent := fmt.Sprintf("projects/%s/locations/%s", t.Source.Project, t.Source.Location)
	req := &dataprocpb.ListBatchesRequest{
		Parent:  parent,
		OrderBy: "create_time desc",
	}

	paramMap := params.AsMap()
	if ps, ok := paramMap["pageSize"]; ok && ps != nil {
		req.PageSize = int32(ps.(int))
		if (req.PageSize) <= 0 {
			return nil, fmt.Errorf("pageSize must be positive: %d", req.PageSize)
		}
	}
	if pt, ok := paramMap["pageToken"]; ok && pt != nil {
		req.PageToken = pt.(string)
	}
	if filter, ok := paramMap["filter"]; ok && filter != nil {
		req.Filter = filter.(string)
	}

	it := client.ListBatches(ctx, req)
	pager := iterator.NewPager(it, int(req.PageSize), req.PageToken)

	var batchPbs []*dataprocpb.Batch
	nextPageToken, err := pager.NextPage(&batchPbs)
	if err != nil {
		return nil, fmt.Errorf("failed to list batches: %w", err)
	}

	batches, err := ToBatches(batchPbs)
	if err != nil {
		return nil, err
	}

	return ListBatchesResponse{Batches: batches, NextPageToken: nextPageToken}, nil
}

// ToBatches converts a slice of protobuf Batch messages to a slice of Batch structs.
func ToBatches(batchPbs []*dataprocpb.Batch) ([]Batch, error) {
	batches := make([]Batch, 0, len(batchPbs))
	for _, batchPb := range batchPbs {
		consoleUrl, err := common.BatchConsoleURLFromProto(batchPb)
		if err != nil {
			return nil, fmt.Errorf("error generating console url: %v", err)
		}
		logsUrl, err := common.BatchLogsURLFromProto(batchPb)
		if err != nil {
			return nil, fmt.Errorf("error generating logs url: %v", err)
		}
		batch := Batch{
			Name:       batchPb.Name,
			UUID:       batchPb.Uuid,
			State:      batchPb.State.Enum().String(),
			Creator:    batchPb.Creator,
			CreateTime: batchPb.CreateTime.AsTime().Format(time.RFC3339),
			Operation:  batchPb.Operation,
			ConsoleURL: consoleUrl,
			LogsURL:    logsUrl,
		}
		batches = append(batches, batch)
	}
	return batches, nil
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

func (t Tool) Authorized(services []string) bool {
	return tools.IsAuthorized(t.AuthRequired, services)
}

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) bool {
	// Client OAuth not supported, rely on ADCs.
	return false
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
