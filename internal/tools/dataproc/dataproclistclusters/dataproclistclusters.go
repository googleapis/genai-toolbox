// Copyright 2026 Google LLC
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

package dataproclistclusters

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/dataproc/v2/apiv1/dataprocpb"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/dataproc"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/dataproc/common"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/api/iterator"
)

const kind = "dataproc-list-clusters"

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
	Type         string   `yaml:"type" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

// ToolConfigType returns the unique name for this tool.
func (cfg Config) ToolConfigType() string {
	return kind
}

// Initialize creates a new Tool instance.
func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("source %q not found", cfg.Source)
	}

	ds, ok := rawS.(*dataproc.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source type must be `%s`", kind, dataproc.SourceType)
	}

	desc := cfg.Description
	if desc == "" {
		desc = "Lists and filters Dataproc clusters"
	}

	allParameters := parameters.Parameters{
		parameters.NewStringParameterWithRequired("filter", `A filter constraining the clusters to list. Filters are case-sensitive and have the following syntax: field = value [AND [field = value]] ...  where field is one of status.state, clusterName, or labels.[KEY], and [KEY] is a label key. value can be * to match all values. status.state can be one of the following: ACTIVE, INACTIVE, CREATING, RUNNING, ERROR, DELETING, UPDATING, STOPPING, or STOPPED. ACTIVE contains the CREATING, UPDATING, and RUNNING states. INACTIVE contains the DELETING, ERROR, STOPPING, and STOPPED states. clusterName is the name of the cluster provided at creation time. Only the logical AND operator is supported; space-separated items are treated as having an implicit AND operator.`, false),
		parameters.NewIntParameterWithDefault("pageSize", 20, "The maximum number of clusters to return in a single page (default 20)"),
		parameters.NewStringParameterWithRequired("pageToken", "A page token, received from a previous `ListClusters` call", false),
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

	Source *dataproc.Source

	manifest    tools.Manifest
	mcpManifest tools.McpManifest
	Parameters  parameters.Parameters
}

// ListClustersResponse is the response from the list clusters API.
type ListClustersResponse struct {
	Clusters      []Cluster `json:"clusters"`
	NextPageToken string    `json:"nextPageToken"`
}

// Cluster represents a single Dataproc cluster.
type Cluster struct {
	Name       string `json:"name"` // Full resource name
	UUID       string `json:"uuid"`
	State      string `json:"state"`
	CreateTime string `json:"createTime"`
	ConsoleURL string `json:"consoleUrl"`
	LogsURL    string `json:"logsUrl"`
}

// Invoke executes the tool's operation.
func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	client := t.Source.GetClusterControllerClient()

	req := &dataprocpb.ListClustersRequest{
		ProjectId: t.Source.Project,
		Region:    t.Source.Region,
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

	it := client.ListClusters(ctx, req)
	pager := iterator.NewPager(it, int(req.PageSize), req.PageToken)

	var clusterPbs []*dataprocpb.Cluster
	nextPageToken, err := pager.NextPage(&clusterPbs)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	clusters, err := ToClusters(clusterPbs, t.Source.Region)
	if err != nil {
		return nil, err
	}

	return ListClustersResponse{Clusters: clusters, NextPageToken: nextPageToken}, nil
}

// ToClusters converts a slice of protobuf Cluster messages to a slice of Cluster structs.
func ToClusters(clusterPbs []*dataprocpb.Cluster, region string) ([]Cluster, error) {
	clusters := make([]Cluster, 0, len(clusterPbs))
	for _, clusterPb := range clusterPbs {
		consoleUrl := common.ClusterConsoleURLFromProto(clusterPb, region)
		logsUrl := common.ClusterLogsURLFromProto(clusterPb, region)

		state := "STATE_UNSPECIFIED"
		// Extract create time from status history.
		var createTime string
		if clusterPb.Status != nil {
			state = clusterPb.Status.State.Enum().String()
			if clusterPb.Status.StateStartTime != nil {
				createTime = clusterPb.Status.StateStartTime.AsTime().Format(time.RFC3339)
			}
		}

		fullName := fmt.Sprintf("projects/%s/regions/%s/clusters/%s", clusterPb.ProjectId, region, clusterPb.ClusterName)

		cluster := Cluster{
			Name:       fullName,
			UUID:       clusterPb.ClusterUuid,
			State:      state,
			CreateTime: createTime,
			ConsoleURL: consoleUrl,
			LogsURL:    logsUrl,
		}
		clusters = append(clusters, cluster)
	}
	return clusters, nil
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

func (t Tool) GetParameters() parameters.Parameters {
	return t.Parameters
}
