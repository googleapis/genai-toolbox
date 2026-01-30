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

package dataproc

import (
	"context"
	"errors"
	"fmt"

	dataproc "cloud.google.com/go/dataproc/v2/apiv1"
	longrunning "cloud.google.com/go/longrunning/autogen"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/api/option"
)

const SourceType string = "dataproc"

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
	Name    string `yaml:"name" validate:"required"`
	Type    string `yaml:"type" validate:"required"`
	Project string `yaml:"project" validate:"required"`
	Region  string `yaml:"region" validate:"required"`
}

func (r Config) SourceConfigType() string {
	return SourceType
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	ua, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error in User Agent retrieval: %s", err)
	}
	endpoint := fmt.Sprintf("%s-dataproc.googleapis.com:443", r.Region)
	client, err := dataproc.NewClusterControllerClient(ctx, option.WithEndpoint(endpoint), option.WithUserAgent(ua))
	if err != nil {
		return nil, fmt.Errorf("failed to create dataproc client: %w", err)
	}
	opsClient, err := longrunning.NewOperationsClient(ctx, option.WithEndpoint(endpoint), option.WithUserAgent(ua))
	if err != nil {
		return nil, fmt.Errorf("failed to create longrunning client: %w", err)
	}
	jobClient, err := dataproc.NewJobControllerClient(ctx, option.WithEndpoint(endpoint), option.WithUserAgent(ua))
	if err != nil {
		return nil, fmt.Errorf("failed to create dataproc job client: %w", err)
	}

	s := &Source{
		Config:    r,
		Client:    client,
		OpsClient: opsClient,
		JobClient: jobClient,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Config
	Client    *dataproc.ClusterControllerClient
	OpsClient *longrunning.OperationsClient
	JobClient *dataproc.JobControllerClient
}

func (s *Source) SourceType() string {
	return SourceType
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func (s *Source) GetClusterControllerClient() *dataproc.ClusterControllerClient {
	return s.Client
}

func (s *Source) GetOperationsClient(ctx context.Context) (*longrunning.OperationsClient, error) {
	return s.OpsClient, nil
}

func (s *Source) GetJobControllerClient() *dataproc.JobControllerClient {
	return s.JobClient
}

func (s *Source) Close() error {
	return errors.Join(s.Client.Close(), s.OpsClient.Close(), s.JobClient.Close())
}
