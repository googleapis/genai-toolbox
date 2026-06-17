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
	"fmt"
	"strings"

	dataplexapi "cloud.google.com/go/dataplex/apiv1"
	"cloud.google.com/go/dataplex/apiv1/dataplexpb"
	"github.com/cenkalti/backoff/v5"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	grpcstatus "google.golang.org/grpc/status"
)

const SourceType string = "dataplex"

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
	client, dataScanClient, dataProductClient, err := initDataplexConnection(ctx, tracer, r.Name, r.Project)
	if err != nil {
		return nil, err
	}
	s := &Source{
		Config:            r,
		Client:            client,
		DataScanClient:    dataScanClient,
		dataProductClient: dataProductClient,
	}

	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Config
	Client            *dataplexapi.CatalogClient
	DataScanClient    *dataplexapi.DataScanClient
	dataProductClient *dataplexapi.DataProductClient
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

func (s *Source) GetDataProductClient() *dataplexapi.DataProductClient {
	return s.dataProductClient
}

func initDataplexConnection(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	project string,
) (*dataplexapi.CatalogClient, *dataplexapi.DataScanClient, *dataplexapi.DataProductClient, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceType, name)
	defer span.End()

	cred, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to find default Google Cloud credentials for project %q: %w", project, err)
	}

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	client, err := dataplexapi.NewCatalogClient(ctx, option.WithUserAgent(userAgent), option.WithCredentials(cred))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create Dataplex client for project %q: %w", project, err)
	}

	dataScanClient, err := dataplexapi.NewDataScanClient(ctx, option.WithUserAgent(userAgent), option.WithCredentials(cred))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create Dataplex DataScan client for project %q: %w", project, err)
	}

	dataProductClient, err := dataplexapi.NewDataProductClient(ctx, option.WithUserAgent(userAgent), option.WithCredentials(cred))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create Dataplex DataProduct client for project %q: %w", project, err)
	}
	return client, dataScanClient, dataProductClient, nil
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

type DataProductSummary struct {
	LocationID    string   `json:"locationId"`
	DataProductID string   `json:"dataProductId"`
	DisplayName   string   `json:"displayName"`
	OwnerEmails   []string `json:"ownerEmails"`
	AssetCount    int32    `json:"assetCount"`
}

func (s *Source) ListDataProducts(
	ctx context.Context,
	filter string,
	pageSize int,
	orderBy string,
) ([]*DataProductSummary, error) {
	if s.GetDataProductClient() == nil {
		return nil, fmt.Errorf("dataplex data product client is not initialized")
	}
	if pageSize <= 0 {
		return nil, fmt.Errorf("pageSize must be positive: %d", pageSize)
	}
	parent := fmt.Sprintf("projects/%s/locations/-", s.ProjectID())
	req := &dataplexpb.ListDataProductsRequest{
		Parent:   parent,
		Filter:   filter,
		PageSize: int32(pageSize),
		OrderBy:  orderBy,
	}

	it := s.GetDataProductClient().ListDataProducts(ctx, req)
	var results []*DataProductSummary

	for len(results) < pageSize {
		dp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if st, ok := grpcstatus.FromError(err); ok {
				return nil, fmt.Errorf("failed to list data products: code=%s message=%s", st.Code(), st.Message())
			}
			return nil, fmt.Errorf("failed to list data products: %w", err)
		}
		parts := strings.Split(dp.GetName(), "/")
		var locationId, dataProductId string
		if len(parts) >= 6 && parts[0] == "projects" && parts[2] == "locations" && parts[4] == "dataProducts" {
			locationId = parts[3]
			dataProductId = parts[5]
		}
		results = append(results, &DataProductSummary{
			LocationID:    locationId,
			DataProductID: dataProductId,
			DisplayName:   dp.GetDisplayName(),
			OwnerEmails:   dp.GetOwnerEmails(),
			AssetCount:    dp.GetAssetCount(),
		})
	}
	return results, nil
}

type AccessGroup struct {
	ID             string `json:"id"`
	DisplayName    string `json:"displayName"`
	Description    string `json:"description"`
	GoogleGroup    string `json:"googleGroup"`
	ServiceAccount string `json:"serviceAccount"`
}

type DataProduct struct {
	LocationID    string            `json:"locationId"`
	DataProductID string            `json:"dataProductId"`
	DisplayName   string            `json:"displayName"`
	Description   string            `json:"description"`
	OwnerEmails   []string          `json:"ownerEmails"`
	AssetCount    int32             `json:"assetCount"`
	Labels        map[string]string `json:"labels"`
	AccessGroups  []AccessGroup     `json:"accessGroups"`
}

func (s *Source) GetDataProduct(ctx context.Context, locationId string, dataProductId string) (*DataProduct, error) {
	if s.GetDataProductClient() == nil {
		return nil, fmt.Errorf("dataplex data product client is not initialized")
	}
	name := fmt.Sprintf("projects/%s/locations/%s/dataProducts/%s", s.ProjectID(), locationId, dataProductId)
	req := &dataplexpb.GetDataProductRequest{
		Name: name,
	}
	resp, err := s.GetDataProductClient().GetDataProduct(ctx, req)
	if err != nil {
		return nil, err
	}

	accessGroups := []AccessGroup{}
	for _, ag := range resp.GetAccessGroups() {
		accessGroups = append(accessGroups, AccessGroup{
			ID:             ag.GetId(),
			DisplayName:    ag.GetDisplayName(),
			Description:    ag.GetDescription(),
			GoogleGroup:    ag.GetPrincipal().GetGoogleGroup(),
			ServiceAccount: ag.GetPrincipal().GetServiceAccount(),
		})
	}

	parts := strings.Split(resp.GetName(), "/")
	var locId, prodId string
	if len(parts) >= 6 && parts[0] == "projects" && parts[2] == "locations" && parts[4] == "dataProducts" {
		locId = parts[3]
		prodId = parts[5]
	}

	return &DataProduct{
		LocationID:    locId,
		DataProductID: prodId,
		DisplayName:   resp.GetDisplayName(),
		Description:   resp.GetDescription(),
		OwnerEmails:   resp.GetOwnerEmails(),
		AssetCount:    resp.GetAssetCount(),
		Labels:        resp.GetLabels(),
		AccessGroups:  accessGroups,
	}, nil
}

type DataAssetSummary struct {
	LocationID    string            `json:"locationId"`
	DataProductID string            `json:"dataProductId"`
	DataAssetID   string            `json:"dataAssetId"`
	ResourceURI   string            `json:"resourceUri"`
	Labels        map[string]string `json:"labels"`
}

func (s *Source) ListDataAssets(
	ctx context.Context,
	locationId string,
	dataProductId string,
	filter string,
	pageSize int,
	orderBy string,
) ([]*DataAssetSummary, error) {
	if s.GetDataProductClient() == nil {
		return nil, fmt.Errorf("dataplex data product client is not initialized")
	}
	if pageSize <= 0 {
		return nil, fmt.Errorf("pageSize must be positive: %d", pageSize)
	}
	parent := fmt.Sprintf("projects/%s/locations/%s/dataProducts/%s", s.ProjectID(), locationId, dataProductId)
	req := &dataplexpb.ListDataAssetsRequest{
		Parent:   parent,
		Filter:   filter,
		PageSize: int32(pageSize),
		OrderBy:  orderBy,
	}

	it := s.GetDataProductClient().ListDataAssets(ctx, req)
	var results []*DataAssetSummary

	for len(results) < pageSize {
		asset, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if st, ok := grpcstatus.FromError(err); ok {
				return nil, fmt.Errorf("failed to list data assets: code=%s message=%s", st.Code(), st.Message())
			}
			return nil, fmt.Errorf("failed to list data assets: %w", err)
		}
		parts := strings.Split(asset.GetName(), "/")
		var locId, prodId, assetId string
		if len(parts) >= 8 && parts[0] == "projects" && parts[2] == "locations" && parts[4] == "dataProducts" && parts[6] == "dataAssets" {
			locId = parts[3]
			prodId = parts[5]
			assetId = parts[7]
		}
		results = append(results, &DataAssetSummary{
			LocationID:    locId,
			DataProductID: prodId,
			DataAssetID:   assetId,
			ResourceURI:   asset.GetResource(),
			Labels:        asset.GetLabels(),
		})
	}
	return results, nil
}

type DataAsset struct {
	LocationID         string                                             `json:"locationId"`
	DataProductID      string                                             `json:"dataProductId"`
	DataAssetID        string                                             `json:"dataAssetId"`
	ResourceURI        string                                             `json:"resourceUri"`
	Labels             map[string]string                                  `json:"labels"`
	AccessGroupConfigs map[string]*dataplexpb.DataAsset_AccessGroupConfig `json:"accessGroupConfigs"`
}

func (s *Source) GetDataAsset(ctx context.Context, locationId string, dataProductId string, dataAssetId string) (*DataAsset, error) {
	if s.GetDataProductClient() == nil {
		return nil, fmt.Errorf("dataplex data product client is not initialized")
	}
	name := fmt.Sprintf("projects/%s/locations/%s/dataProducts/%s/dataAssets/%s", s.ProjectID(), locationId, dataProductId, dataAssetId)
	req := &dataplexpb.GetDataAssetRequest{
		Name: name,
	}
	resp, err := s.GetDataProductClient().GetDataAsset(ctx, req)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(resp.GetName(), "/")
	var locId, prodId, assetId string
	if len(parts) >= 8 && parts[0] == "projects" && parts[2] == "locations" && parts[4] == "dataProducts" && parts[6] == "dataAssets" {
		locId = parts[3]
		prodId = parts[5]
		assetId = parts[7]
	}

	return &DataAsset{
		LocationID:         locId,
		DataProductID:      prodId,
		DataAssetID:        assetId,
		ResourceURI:        resp.GetResource(),
		Labels:             resp.GetLabels(),
		AccessGroupConfigs: resp.GetAccessGroupConfigs(),
	}, nil
}
