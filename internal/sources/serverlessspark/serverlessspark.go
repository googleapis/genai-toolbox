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

package serverlessspark

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	dataproc "cloud.google.com/go/dataproc/v2/apiv1"
	"cloud.google.com/go/dataproc/v2/apiv1/dataprocpb"
	longrunning "cloud.google.com/go/longrunning/autogen"
	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/encoding/protojson"
)

const SourceType string = "serverless-spark"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceType, newConfig) {
		panic(fmt.Sprintf("source type %q already registered", SourceType))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name     string `yaml:"name" validate:"required"`
	Type     string `yaml:"type" validate:"required"`
	Project  string `yaml:"project" validate:"required"`
	Location string `yaml:"location" validate:"required"`
}

func (r Config) SourceConfigType() string {
	return SourceType
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	ua, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error in User Agent retrieval: %s", err)
	}
	endpoint := fmt.Sprintf("%s-dataproc.googleapis.com:443", r.Location)
	client, err := dataproc.NewBatchControllerClient(ctx, option.WithEndpoint(endpoint), option.WithUserAgent(ua))
	if err != nil {
		return nil, fmt.Errorf("failed to create dataproc client: %w", err)
	}
	opsClient, err := longrunning.NewOperationsClient(ctx, option.WithEndpoint(endpoint), option.WithUserAgent(ua))
	if err != nil {
		return nil, fmt.Errorf("failed to create longrunning client: %w", err)
	}

	s := &Source{
		Config:    r,
		Client:    client,
		OpsClient: opsClient,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Config
	Client    *dataproc.BatchControllerClient
	OpsClient *longrunning.OperationsClient
}

func (s *Source) SourceType() string {
	return SourceType
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func (s *Source) GetProject() string {
	return s.Project
}

func (s *Source) GetLocation() string {
	return s.Location
}

func (s *Source) GetBatchControllerClient() *dataproc.BatchControllerClient {
	return s.Client
}

func (s *Source) GetOperationsClient(ctx context.Context) (*longrunning.OperationsClient, error) {
	return s.OpsClient, nil
}

func (s *Source) Close() error {
	if err := s.Client.Close(); err != nil {
		return err
	}
	if err := s.OpsClient.Close(); err != nil {
		return err
	}
	return nil
}

func (s *Source) CancelOperation(ctx context.Context, operation string) (any, error) {
	req := &longrunningpb.CancelOperationRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", s.GetProject(), s.GetLocation(), operation),
	}
	client, err := s.GetOperationsClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get operations client: %w", err)
	}
	err = client.CancelOperation(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel operation: %w", err)
	}
	return fmt.Sprintf("Cancelled [%s].", operation), nil
}

func (s *Source) CreateBatch(ctx context.Context, batch *dataprocpb.Batch) (map[string]any, error) {
	req := &dataprocpb.CreateBatchRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", s.GetProject(), s.GetLocation()),
		Batch:  batch,
	}

	client := s.GetBatchControllerClient()
	op, err := client.CreateBatch(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create batch: %w", err)
	}
	meta, err := op.Metadata()
	if err != nil {
		return nil, fmt.Errorf("failed to get create batch op metadata: %w", err)
	}

	projectID, location, batchID, err := ExtractBatchDetails(meta.GetBatch())
	if err != nil {
		return nil, fmt.Errorf("error extracting batch details from name %q: %v", meta.GetBatch(), err)
	}
	consoleUrl := BatchConsoleURL(projectID, location, batchID)
	logsUrl := BatchLogsURL(projectID, location, batchID, meta.GetCreateTime().AsTime(), time.Time{})

	wrappedResult := map[string]any{
		"opMetadata": meta,
		"consoleUrl": consoleUrl,
		"logsUrl":    logsUrl,
	}
	return wrappedResult, nil
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

func (s *Source) ListBatches(ctx context.Context, ps *int, pt, filter string) (any, error) {
	client := s.GetBatchControllerClient()
	parent := fmt.Sprintf("projects/%s/locations/%s", s.GetProject(), s.GetLocation())
	req := &dataprocpb.ListBatchesRequest{
		Parent:  parent,
		OrderBy: "create_time desc",
	}

	if ps != nil {
		req.PageSize = int32(*ps)
	}
	if pt != "" {
		req.PageToken = pt
	}
	if filter != "" {
		req.Filter = filter
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
		consoleUrl, err := BatchConsoleURLFromProto(batchPb)
		if err != nil {
			return nil, fmt.Errorf("error generating console url: %v", err)
		}
		logsUrl, err := BatchLogsURLFromProto(batchPb)
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

func (s *Source) GetBatch(ctx context.Context, name string) (map[string]any, error) {
	client := s.GetBatchControllerClient()
	req := &dataprocpb.GetBatchRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/batches/%s", s.GetProject(), s.GetLocation(), name),
	}

	batchPb, err := client.GetBatch(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch: %w", err)
	}

	jsonBytes, err := protojson.Marshal(batchPb)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch to JSON: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch JSON: %w", err)
	}

	consoleUrl, err := BatchConsoleURLFromProto(batchPb)
	if err != nil {
		return nil, fmt.Errorf("error generating console url: %v", err)
	}
	logsUrl, err := BatchLogsURLFromProto(batchPb)
	if err != nil {
		return nil, fmt.Errorf("error generating logs url: %v", err)
	}

	wrappedResult := map[string]any{
		"consoleUrl": consoleUrl,
		"logsUrl":    logsUrl,
		"batch":      result,
	}

	return wrappedResult, nil
}
