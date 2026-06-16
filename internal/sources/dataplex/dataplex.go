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

package dataplex

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	dataplexapi "cloud.google.com/go/dataplex/apiv1"
	"cloud.google.com/go/dataplex/apiv1/dataplexpb"
	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/cenkalti/backoff/v5"
	"github.com/goccy/go-yaml"
	"github.com/google/uuid"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

const SourceType string = "dataplex"

var operationNameRegex = regexp.MustCompile(`^projects/[^/]+/locations/[^/]+/operations/[^/]+$`)

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
	// Dataplex configs
	Name    string `yaml:"name" validate:"required"`
	Type    string `yaml:"type" validate:"required"`
	Project string `yaml:"project" validate:"required"`
}

func (r Config) SourceConfigType() string {
	// Returns Dataplex source type
	return SourceType
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	// Initializes a Dataplex source
	client, dataScanClient, err := initDataplexConnection(ctx, tracer, r.Name, r.Project)
	if err != nil {
		return nil, err
	}
	s := &Source{
		Config:         r,
		Client:         client,
		DataScanClient: dataScanClient,
	}

	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Config
	Client         *dataplexapi.CatalogClient
	DataScanClient *dataplexapi.DataScanClient
}

func (s *Source) SourceType() string {
	// Returns Dataplex source type
	return SourceType
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func (s *Source) ProjectID() string {
	return s.Project
}

func (s *Source) CatalogClient() *dataplexapi.CatalogClient {
	return s.Client
}

func (s *Source) GetDataScanClient() *dataplexapi.DataScanClient {
	return s.DataScanClient
}

func initDataplexConnection(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	project string,
) (*dataplexapi.CatalogClient, *dataplexapi.DataScanClient, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceType, name)
	defer span.End()

	cred, err := google.FindDefaultCredentials(ctx, sources.CloudPlatformScope)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find default Google Cloud credentials for project %q: %w", project, err)
	}

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	client, err := dataplexapi.NewCatalogClient(ctx, option.WithUserAgent(userAgent), option.WithCredentials(cred))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Dataplex client for project %q: %w", project, err)
	}

	dataScanClient, err := dataplexapi.NewDataScanClient(ctx, option.WithUserAgent(userAgent), option.WithCredentials(cred))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Dataplex DataScan client for project %q: %w", project, err)
	}
	return client, dataScanClient, nil
}

func (s *Source) LookupEntry(ctx context.Context, name string, view int, aspectTypes []string, entry string) (*dataplexpb.Entry, error) {
	viewMap := map[int]dataplexpb.EntryView{
		1: dataplexpb.EntryView_BASIC,
		2: dataplexpb.EntryView_FULL,
		3: dataplexpb.EntryView_CUSTOM,
		4: dataplexpb.EntryView_ALL,
	}
	req := &dataplexpb.LookupEntryRequest{
		Name:        name,
		View:        viewMap[view],
		AspectTypes: aspectTypes,
		Entry:       entry,
	}
	result, err := s.CatalogClient().LookupEntry(ctx, req)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Source) searchRequest(ctx context.Context, query string, pageSize int, orderBy string, scope string) (*dataplexapi.SearchEntriesResultIterator, error) {
	// Create SearchEntriesRequest with the provided parameters
	req := &dataplexpb.SearchEntriesRequest{
		Query:          query,
		Name:           fmt.Sprintf("projects/%s/locations/global", s.ProjectID()),
		PageSize:       int32(pageSize),
		OrderBy:        orderBy,
		SemanticSearch: true,
	}

	if scope != "" {
		req.Scope = scope
	}

	// Perform the search using the CatalogClient - this will return an iterator
	it := s.CatalogClient().SearchEntries(ctx, req)
	if it == nil {
		return nil, fmt.Errorf("failed to create search entries iterator for project %q", s.ProjectID())
	}
	return it, nil
}

func (s *Source) SearchAspectTypes(ctx context.Context, query string, pageSize int, orderBy string) ([]*dataplexpb.AspectType, error) {
	if pageSize <= 0 {
		return nil, fmt.Errorf("pageSize must be positive: %d", pageSize)
	}
	q := query + " type=projects/dataplex-types/locations/global/entryTypes/aspecttype"
	it, err := s.searchRequest(ctx, q, pageSize, orderBy, "")
	if err != nil {
		return nil, err
	}

	// Iterate through the search results and call GetAspectType for each result using the resource name
	var results []*dataplexpb.AspectType
	for len(results) < pageSize {
		entry, err := it.Next()

		if err == iterator.Done {
			break
		}
		if err != nil {
			if st, ok := grpcstatus.FromError(err); ok {
				errorCode := st.Code()
				errorMessage := st.Message()
				return nil, fmt.Errorf("failed to search aspect types with error code: %q message: %s", errorCode.String(), errorMessage)
			}
			return nil, fmt.Errorf("failed to search aspect types: %w", err)
		}

		// Create an instance of exponential backoff with default values for retrying GetAspectType calls
		// InitialInterval, RandomizationFactor, Multiplier, MaxInterval = 500 ms, 0.5, 1.5, 60 s
		getAspectBackOff := backoff.NewExponentialBackOff()

		resourceName := entry.DataplexEntry.GetEntrySource().Resource
		getAspectTypeReq := &dataplexpb.GetAspectTypeRequest{
			Name: resourceName,
		}

		operation := func() (*dataplexpb.AspectType, error) {
			aspectType, err := s.CatalogClient().GetAspectType(ctx, getAspectTypeReq)
			if err != nil {
				return nil, fmt.Errorf("failed to get aspect type for entry %q: %w", resourceName, err)
			}
			return aspectType, nil
		}

		// Retry the GetAspectType operation with exponential backoff
		aspectType, err := backoff.Retry(ctx, operation, backoff.WithBackOff(getAspectBackOff))
		if err != nil {
			return nil, fmt.Errorf("failed to get aspect type after retries for entry %q: %w", resourceName, err)
		}

		results = append(results, aspectType)
	}
	return results, nil
}

func (s *Source) SearchEntries(ctx context.Context, query string, pageSize int, orderBy string, scope string) ([]*dataplexpb.SearchEntriesResult, error) {
	if pageSize <= 0 {
		return nil, fmt.Errorf("pageSize must be positive: %d", pageSize)
	}
	it, err := s.searchRequest(ctx, query, pageSize, orderBy, scope)
	if err != nil {
		return nil, err
	}

	var results []*dataplexpb.SearchEntriesResult
	for len(results) < pageSize {
		entry, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if st, ok := grpcstatus.FromError(err); ok {
				errorCode := st.Code()
				errorMessage := st.Message()
				return nil, fmt.Errorf("failed to search entries with error code: %q message: %s", errorCode.String(), errorMessage)
			}
			return nil, fmt.Errorf("failed to search entries: %w", err)
		}
		results = append(results, entry)
	}
	return results, nil
}

func (s *Source) LookupContext(ctx context.Context, name string, resources []string) (*dataplexpb.LookupContextResponse, error) {
	req := &dataplexpb.LookupContextRequest{
		Name:      name,
		Resources: resources,
	}
	result, err := s.CatalogClient().LookupContext(ctx, req)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Source) SearchDataQualityScans(ctx context.Context, filter string, pageSize int, orderBy string) ([]*dataplexpb.DataScan, error) {
	if pageSize <= 0 {
		return nil, fmt.Errorf("pageSize must be positive: %d", pageSize)
	}
	req := &dataplexpb.ListDataScansRequest{
		Parent:   fmt.Sprintf("projects/%s/locations/-", s.ProjectID()),
		Filter:   filter,
		PageSize: int32(pageSize),
		OrderBy:  orderBy,
	}

	it := s.GetDataScanClient().ListDataScans(ctx, req)

	var results []*dataplexpb.DataScan
	for len(results) < pageSize {
		scan, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if st, ok := grpcstatus.FromError(err); ok {
				return nil, fmt.Errorf("failed to list data scans: code=%s message=%s", st.Code(), st.Message())
			}
			return nil, fmt.Errorf("failed to list data scans: %w", err)
		}
		results = append(results, scan)
	}
	return results, nil
}

func (s *Source) GenerateDataInsights(ctx context.Context, location, resourcePath string, publish bool) (string, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", s.ProjectID(), location)
	dataScanID := fmt.Sprintf("nq-doc-%s", uuid.New().String())

	req := &dataplexpb.CreateDataScanRequest{
		Parent:     parent,
		DataScanId: dataScanID,
		DataScan: &dataplexpb.DataScan{
			Data: &dataplexpb.DataSource{
				Source: &dataplexpb.DataSource_Resource{
					Resource: resourcePath,
				},
			},
			Spec: &dataplexpb.DataScan_DataDocumentationSpec{
				DataDocumentationSpec: &dataplexpb.DataDocumentationSpec{
					CatalogPublishingEnabled: publish,
				},
			},
			ExecutionSpec: &dataplexpb.DataScan_ExecutionSpec{
				Trigger: &dataplexpb.Trigger{
					Mode: &dataplexpb.Trigger_OneTime_{
						OneTime: &dataplexpb.Trigger_OneTime{},
					},
				},
			},
			Type: dataplexpb.DataScanType_DATA_DOCUMENTATION,
			Labels: map[string]string{
				"onemcp-server": "true",
			},
		},
	}

	op, err := s.DataScanClient.CreateDataScan(ctx, req)
	if err != nil {
		return "", err
	}
	return op.Name(), nil
}

func (s *Source) GetDataScan(ctx context.Context, location, scanID string) (*dataplexpb.DataScan, error) {
	name := fmt.Sprintf("projects/%s/locations/%s/dataScans/%s", s.ProjectID(), location, scanID)
	req := &dataplexpb.GetDataScanRequest{
		Name: name,
		View: dataplexpb.GetDataScanRequest_FULL,
	}
	return s.DataScanClient.GetDataScan(ctx, req)
}

func (s *Source) GetOperation(ctx context.Context, opName string) (map[string]any, error) {
	if !operationNameRegex.MatchString(opName) {
		return nil, fmt.Errorf("invalid operation name format: %q (expected projects/*/locations/*/operations/*)", opName)
	}

	req := &longrunningpb.GetOperationRequest{
		Name: opName,
	}
	op, err := s.DataScanClient.LROClient.GetOperation(ctx, req)
	if err != nil {
		return nil, err
	}

	bytes, err := protojson.Marshal(op)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal operation to JSON: %w", err)
	}

	var opData map[string]any
	if err := json.Unmarshal(bytes, &opData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operation JSON to map: %w", err)
	}

	return opData, nil
}

func (s *Source) GetJobStatus(ctx context.Context, location, scanID, jobID string) (*dataplexpb.DataScanJob, error) {
	// If jobID is provided, fetch that specific job directly!
	if jobID != "" {
		name := fmt.Sprintf("projects/%s/locations/%s/dataScans/%s/jobs/%s", s.ProjectID(), location, scanID, jobID)
		req := &dataplexpb.GetDataScanJobRequest{
			Name: name,
		}
		return s.DataScanClient.GetDataScanJob(ctx, req)
	}

	// Fallback to listing and returning the latest job (PageSize: 1)
	parent := fmt.Sprintf("projects/%s/locations/%s/dataScans/%s", s.ProjectID(), location, scanID)
	req := &dataplexpb.ListDataScanJobsRequest{
		Parent:   parent,
		PageSize: 1,
	}

	it := s.DataScanClient.ListDataScanJobs(ctx, req)
	if it == nil {
		return nil, fmt.Errorf("failed to list data scan jobs for scan %q", scanID)
	}

	job, err := it.Next()
	if err == iterator.Done {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (s *Source) GenerateDataProfile(ctx context.Context, location, resourcePath string, publish bool) (string, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", s.ProjectID(), location)
	dataScanID := fmt.Sprintf("nq-prof-%s", uuid.New().String())

	req := &dataplexpb.CreateDataScanRequest{
		Parent:     parent,
		DataScanId: dataScanID,
		DataScan: &dataplexpb.DataScan{
			Data: &dataplexpb.DataSource{
				Source: &dataplexpb.DataSource_Resource{
					Resource: resourcePath,
				},
			},
			Spec: &dataplexpb.DataScan_DataProfileSpec{
				DataProfileSpec: &dataplexpb.DataProfileSpec{
					CatalogPublishingEnabled: publish,
				},
			},
			ExecutionSpec: &dataplexpb.DataScan_ExecutionSpec{
				Trigger: &dataplexpb.Trigger{
					Mode: &dataplexpb.Trigger_OneTime_{
						OneTime: &dataplexpb.Trigger_OneTime{},
					},
				},
			},
			Type: dataplexpb.DataScanType_DATA_PROFILE,
			Labels: map[string]string{
				"onemcp-server": "true",
			},
		},
	}

	op, err := s.DataScanClient.CreateDataScan(ctx, req)
	if err != nil {
		return "", err
	}
	return op.Name(), nil
}

func (s *Source) GenerateDataDiscovery(ctx context.Context, location, resourcePath string) (string, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", s.ProjectID(), location)
	dataScanID := fmt.Sprintf("nq-disc-%s", uuid.New().String())

	req := &dataplexpb.CreateDataScanRequest{
		Parent:     parent,
		DataScanId: dataScanID,
		DataScan: &dataplexpb.DataScan{
			Data: &dataplexpb.DataSource{
				Source: &dataplexpb.DataSource_Resource{
					Resource: resourcePath,
				},
			},
			Spec: &dataplexpb.DataScan_DataDiscoverySpec{
				DataDiscoverySpec: &dataplexpb.DataDiscoverySpec{},
			},
			ExecutionSpec: &dataplexpb.DataScan_ExecutionSpec{
				Trigger: &dataplexpb.Trigger{
					Mode: &dataplexpb.Trigger_OneTime_{
						OneTime: &dataplexpb.Trigger_OneTime{},
					},
				},
			},
			Type: dataplexpb.DataScanType_DATA_DISCOVERY,
			Labels: map[string]string{
				"onemcp-server": "true",
			},
		},
	}

	op, err := s.DataScanClient.CreateDataScan(ctx, req)
	if err != nil {
		return "", err
	}
	return op.Name(), nil
}

func (s *Source) GenerateDataQuality(ctx context.Context, location, resourcePath string, specJSON string, publish bool) (string, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", s.ProjectID(), location)
	dataScanID := fmt.Sprintf("nq-dq-%s", uuid.New().String())

	var dqSpec dataplexpb.DataQualitySpec
	if err := protojson.Unmarshal([]byte(specJSON), &dqSpec); err != nil {
		return "", fmt.Errorf("failed to parse data quality spec JSON: %w", err)
	}
	dqSpec.CatalogPublishingEnabled = publish

	req := &dataplexpb.CreateDataScanRequest{
		Parent:     parent,
		DataScanId: dataScanID,
		DataScan: &dataplexpb.DataScan{
			Data: &dataplexpb.DataSource{
				Source: &dataplexpb.DataSource_Resource{
					Resource: resourcePath,
				},
			},
			Spec: &dataplexpb.DataScan_DataQualitySpec{
				DataQualitySpec: &dqSpec,
			},
			ExecutionSpec: &dataplexpb.DataScan_ExecutionSpec{
				Trigger: &dataplexpb.Trigger{
					Mode: &dataplexpb.Trigger_OneTime_{
						OneTime: &dataplexpb.Trigger_OneTime{},
					},
				},
			},
			Type: dataplexpb.DataScanType_DATA_QUALITY,
			Labels: map[string]string{
				"onemcp-server": "true",
			},
		},
	}

	op, err := s.DataScanClient.CreateDataScan(ctx, req)
	if err != nil {
		return "", err
	}
	return op.Name(), nil
}
