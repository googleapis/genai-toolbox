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

	dataplexapi "cloud.google.com/go/dataplex/apiv1"
	"cloud.google.com/go/dataplex/apiv1/dataplexpb"
	"github.com/cenkalti/backoff/v5"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
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
	client, err := initDataplexConnection(ctx, tracer, r.Name, r.Project)
	if err != nil {
		return nil, err
	}
	s := &Source{
		Config: r,
		Client: client,
	}

	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Config
	Client *dataplexapi.CatalogClient
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

func initDataplexConnection(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	project string,
) (*dataplexapi.CatalogClient, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceType, name)
	defer span.End()

	cred, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find default Google Cloud credentials: %w", err)
	}

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}

	client, err := dataplexapi.NewCatalogClient(ctx, option.WithUserAgent(userAgent), option.WithCredentials(cred))
	if err != nil {
		return nil, fmt.Errorf("failed to create Dataplex client for project %q: %w", project, err)
	}
	return client, nil
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

func (s *Source) searchRequest(ctx context.Context, query string, pageSize int, orderBy string) (*dataplexapi.SearchEntriesResultIterator, error) {
	// Create SearchEntriesRequest with the provided parameters
	req := &dataplexpb.SearchEntriesRequest{
		Query:          query,
		Name:           fmt.Sprintf("projects/%s/locations/global", s.ProjectID()),
		PageSize:       int32(pageSize),
		OrderBy:        orderBy,
		SemanticSearch: true,
	}

	// Perform the search using the CatalogClient - this will return an iterator
	it := s.CatalogClient().SearchEntries(ctx, req)
	if it == nil {
		return nil, fmt.Errorf("failed to create search entries iterator for project %q", s.ProjectID())
	}
	return it, nil
}

func (s *Source) SearchAspectTypes(ctx context.Context, query string, pageSize int, orderBy string) ([]*dataplexpb.AspectType, error) {
	q := query + " type=projects/dataplex-types/locations/global/entryTypes/aspecttype"
	it, err := s.searchRequest(ctx, q, pageSize, orderBy)
	if err != nil {
		return nil, err
	}

	// Iterate through the search results and call GetAspectType for each result using the resource name
	var results []*dataplexpb.AspectType
	for {
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

func (s *Source) SearchEntries(ctx context.Context, query string, pageSize int, orderBy string) ([]*dataplexpb.SearchEntriesResult, error) {
	it, err := s.searchRequest(ctx, query, pageSize, orderBy)
	if err != nil {
		return nil, err
	}

	var results []*dataplexpb.SearchEntriesResult
	for {
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
