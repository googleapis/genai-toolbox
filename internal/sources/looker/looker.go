// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package looker

import (
	"context"
	"fmt"
	"time"

	geminidataanalytics "cloud.google.com/go/geminidataanalytics/apiv1beta"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const SourceKind string = "looker"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{
		Name:               name,
		SslVerification:    true,
		Timeout:            "600s",
		UseClientOAuth:     false,
		ShowHiddenModels:   true,
		ShowHiddenExplores: true,
		ShowHiddenFields:   true,
		Location:           "us",
		SessionLength:      1200,
	} // Default Ssl,timeout, ShowHidden
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name               string `yaml:"name" validate:"required"`
	Kind               string `yaml:"kind" validate:"required"`
	BaseURL            string `yaml:"base_url" validate:"required"`
	ClientId           string `yaml:"client_id"`
	ClientSecret       string `yaml:"client_secret"`
	SslVerification    bool   `yaml:"verify_ssl"`
	UseClientOAuth     bool   `yaml:"use_client_oauth"`
	Timeout            string `yaml:"timeout"`
	ShowHiddenModels   bool   `yaml:"show_hidden_models"`
	ShowHiddenExplores bool   `yaml:"show_hidden_explores"`
	ShowHiddenFields   bool   `yaml:"show_hidden_fields"`
	Project            string `yaml:"project"`
	Location           string `yaml:"location"`
	SessionLength      int64  `yaml:"sessionLength"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

// Initialize initializes a Looker Source instance.
func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}

	duration, err := time.ParseDuration(r.Timeout)
	if err != nil {
		return nil, fmt.Errorf("unable to parse Timeout string as time.Duration: %s", err)
	}

	if !r.SslVerification {
		logger.WarnContext(ctx, "Insecure HTTP is enabled for Looker source %s. TLS certificate verification is skipped.\n", r.Name)
	}
	cfg := rtl.ApiSettings{
		AgentTag:     userAgent,
		BaseUrl:      r.BaseURL,
		ApiVersion:   "4.0",
		VerifySsl:    r.SslVerification,
		Timeout:      int32(duration.Seconds()),
		ClientId:     r.ClientId,
		ClientSecret: r.ClientSecret,
	}

	var tokenSource oauth2.TokenSource
	tokenSource, _ = initGoogleCloudConnection(ctx)

	s := &Source{
		Config:      r,
		ApiSettings: &cfg,
		TokenSource: tokenSource,
	}

	if !r.UseClientOAuth {
		if r.ClientId == "" || r.ClientSecret == "" {
			return nil, fmt.Errorf("client_id and client_secret need to be specified")
		}
		s.Client = v4.NewLookerSDK(rtl.NewAuthSession(cfg))
		resp, err := s.Client.Me("", s.ApiSettings)
		if err != nil {
			return nil, fmt.Errorf("incorrect settings: %w", err)
		}
		logger.DebugContext(ctx, fmt.Sprintf("logged in as %s %s", *resp.FirstName, *resp.LastName))
	}

	return s, nil

}

var _ sources.Source = &Source{}

type Source struct {
	Config
	Client      *v4.LookerSDK
	ApiSettings *rtl.ApiSettings
	TokenSource oauth2.TokenSource
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func (s *Source) GetApiSettings() *rtl.ApiSettings {
	return s.ApiSettings
}

func (s *Source) UseClientAuthorization() bool {
	return s.UseClientOAuth
}

func (s *Source) GoogleCloudProject() string {
	return s.Project
}

func (s *Source) GoogleCloudLocation() string {
	return s.Location
}

func (s *Source) GoogleCloudTokenSource() oauth2.TokenSource {
	return s.TokenSource
}

func (s *Source) GoogleCloudTokenSourceWithScope(ctx context.Context, scope string) (oauth2.TokenSource, error) {
	return google.DefaultTokenSource(ctx, scope)
}

func initGoogleCloudConnection(ctx context.Context) (oauth2.TokenSource, error) {
	cred, err := google.FindDefaultCredentials(ctx, geminidataanalytics.DefaultAuthScopes()...)
	if err != nil {
		return nil, fmt.Errorf("failed to find default Google Cloud credentials with scope %q: %w", geminidataanalytics.DefaultAuthScopes(), err)
	}

	return cred.TokenSource, nil
}
