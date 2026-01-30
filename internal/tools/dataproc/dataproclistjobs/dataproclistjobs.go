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

package dataproclistjobs

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

const kind = "dataproc-list-jobs"

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
		desc = "Lists and filters Dataproc jobs"
	}

	allParameters := parameters.Parameters{
		parameters.NewStringParameterWithRequired("filter", `A filter constraining the jobs to list. Filters are case-sensitive and have the following syntax: field = value [AND [field = value]] ... where field is clusterName, status.state, or labels.[KEY], and [KEY] is a label key. value can be * to match all values. status.state can be one of the following: PENDING, RUNNING, CANCEL_PENDING, JOB_STATE_CANCELLED, DONE, ERROR, or ATTEMPT_FAILURE. Only the logical AND operator is supported; space-separated items are treated as having an implicit AND operator. Filtering by clusterName is recommended to improve query performance.`, false),
		parameters.NewStringParameterWithRequired("jobStateMatcher", "Specifies if the job state matcher should match ALL jobs, only ACTIVE jobs, or only NON_ACTIVE jobs. Defaults to ALL. Supported values: ALL, ACTIVE, NON_ACTIVE.", false),
		parameters.NewIntParameterWithDefault("pageSize", 20, "The maximum number of jobs to return in a single page (default 20)"),
		parameters.NewStringParameterWithRequired("pageToken", "A page token, received from a previous `ListJobs` call", false),
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

// ListJobsResponse is the response from the list jobs API.
type ListJobsResponse struct {
	Jobs          []Job  `json:"jobs"`
	NextPageToken string `json:"nextPageToken"`
}

// Job represents a single Dataproc job.
type Job struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	SubStatus   string `json:"subStatus,omitempty"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime,omitempty"`
	ClusterName string `json:"clusterName"`
	ConsoleURL  string `json:"consoleUrl"`
	LogsURL     string `json:"logsUrl"`
}

// Invoke executes the tool's operation.
func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	client := t.Source.GetJobControllerClient()

	req := &dataprocpb.ListJobsRequest{
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
	if matcher, ok := paramMap["jobStateMatcher"]; ok && matcher != nil {
		matcherStr := matcher.(string)
		if v, ok := dataprocpb.ListJobsRequest_JobStateMatcher_value[matcherStr]; ok {
			req.JobStateMatcher = dataprocpb.ListJobsRequest_JobStateMatcher(v)
		} else {
			return nil, fmt.Errorf("invalid jobStateMatcher: %s. Supported values: ALL, ACTIVE, NON_ACTIVE", matcherStr)
		}
	}

	it := client.ListJobs(ctx, req)
	pager := iterator.NewPager(it, int(req.PageSize), req.PageToken)

	var jobPbs []*dataprocpb.Job
	nextPageToken, err := pager.NextPage(&jobPbs)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	jobs, err := ToJobs(jobPbs, t.Source.Region)
	if err != nil {
		return nil, err
	}

	return ListJobsResponse{Jobs: jobs, NextPageToken: nextPageToken}, nil
}

// ToJobs converts a slice of protobuf Job messages to a slice of Job structs.
func ToJobs(jobPbs []*dataprocpb.Job, region string) ([]Job, error) {
	jobs := make([]Job, 0, len(jobPbs))
	for _, jobPb := range jobPbs {
		consoleUrl := common.JobConsoleURLFromProto(jobPb, region)
		logsUrl, err := common.JobLogsURLFromProto(jobPb, region)
		if err != nil {
			return nil, fmt.Errorf("error generating logs url: %v", err)
		}

		status := "STATE_UNSPECIFIED"
		subStatus := ""
		var startTime, endTime string

		if jobPb.Status != nil {
			status = jobPb.Status.State.Enum().String()
			subStatus = jobPb.Status.Substate.Enum().String()
		}

		var sTime, eTime time.Time
		for _, s := range jobPb.StatusHistory {
			t := s.StateStartTime.AsTime()
			if sTime.IsZero() || t.Before(sTime) {
				sTime = t
			}
		}
		if jobPb.Status != nil {
			t := jobPb.Status.StateStartTime.AsTime()
			if sTime.IsZero() || t.Before(sTime) {
				sTime = t
			}
			switch jobPb.Status.State {
			case dataprocpb.JobStatus_DONE, dataprocpb.JobStatus_CANCELLED, dataprocpb.JobStatus_ERROR:
				eTime = t
			}
		}

		if !sTime.IsZero() {
			startTime = sTime.Format(time.RFC3339)
		}
		if !eTime.IsZero() {
			endTime = eTime.Format(time.RFC3339)
		}

		clusterName := ""
		if jobPb.Placement != nil {
			clusterName = jobPb.Placement.ClusterName
		}

		job := Job{
			ID:          jobPb.Reference.JobId,
			Status:      status,
			SubStatus:   subStatus,
			StartTime:   startTime,
			EndTime:     endTime,
			ClusterName: clusterName,
			ConsoleURL:  consoleUrl,
			LogsURL:     logsUrl,
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
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
