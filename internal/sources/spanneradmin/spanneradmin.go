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

package spanneradmin

import (
	"context"
	"fmt"

	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

const SourceKind string = "spanner-admin"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
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
	Name           string `yaml:"name" validate:"required"`
	Kind           string `yaml:"kind" validate:"required"`
	DefaultProject string `yaml:"defaultProject"`
	UseClientOAuth bool   `yaml:"useClientOAuth"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

// Initialize initializes a Spanner Admin Source instance.
func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	var client *instance.InstanceAdminClient

	if !r.UseClientOAuth {
		ua, err := util.UserAgentFromContext(ctx)
		if err != nil {
			return nil, fmt.Errorf("error in User Agent retrieval: %s", err)
		}
		// Use Application Default Credentials
		client, err = instance.NewInstanceAdminClient(ctx, option.WithUserAgent(ua))
		if err != nil {
			return nil, fmt.Errorf("error creating new spanner instance admin client: %w", err)
		}
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
	Client *instance.InstanceAdminClient
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func (s *Source) GetDefaultProject() string {
	return s.DefaultProject
}

func (s *Source) GetClient(ctx context.Context, accessToken string) (*instance.InstanceAdminClient, error) {
	if s.UseClientOAuth {
		token := &oauth2.Token{AccessToken: accessToken}
		ua, err := util.UserAgentFromContext(ctx)
		if err != nil {
			return nil, err
		}
		client, err := instance.NewInstanceAdminClient(ctx, option.WithTokenSource(oauth2.StaticTokenSource(token)), option.WithUserAgent(ua))
		if err != nil {
			return nil, fmt.Errorf("error creating new spanner instance admin client: %w", err)
		}
		return client, nil
	}
	return s.Client, nil
}

func (s *Source) UseClientAuthorization() bool {
	return s.UseClientOAuth
}
