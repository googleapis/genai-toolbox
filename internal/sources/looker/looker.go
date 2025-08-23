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

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"

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
	actual := Config{Name: name, SslVerification: "true", Timeout: "600s"} // Default Ssl,timeout
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name               string `yaml:"name" validate:"required"`
	Kind               string `yaml:"kind" validate:"required"`
	BaseURL            string `yaml:"base_url" validate:"required"`
	ClientId           string `yaml:"client_id" validate:"required"`
	ClientSecret       string `yaml:"client_secret" validate:"required"`
	SslVerification    string `yaml:"verify_ssl"`
	Timeout            string `yaml:"timeout"`
	ShowHiddenModels   string `yaml:"show_hidden_models"`
	ShowHiddenExplores string `yaml:"show_hidden_explores"`
	ShowHiddenFields   string `yaml:"show_hidden_fields"`
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

	if r.SslVerification != "true" {
		logger.WarnContext(ctx, "Insecure HTTP is enabled for Looker source %s. TLS certificate verification is skipped.\n", r.Name)
	}
	cfg := rtl.ApiSettings{
		AgentTag:     userAgent,
		BaseUrl:      r.BaseURL,
		ApiVersion:   "4.0",
		VerifySsl:    (r.SslVerification == "true"),
		Timeout:      int32(duration.Seconds()),
		ClientId:     r.ClientId,
		ClientSecret: r.ClientSecret,
	}

	sdk := v4.NewLookerSDK(rtl.NewAuthSession(cfg))
	me, err := sdk.Me("", &cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to log in: %s", err)
	}
	logger.DebugContext(ctx, fmt.Sprintf("logged in as user %v %v.\n", *me.FirstName, *me.LastName))

	s := &Source{
		Name:               r.Name,
		Kind:               SourceKind,
		Timeout:            r.Timeout,
		Client:             sdk,
		ApiSettings:        &cfg,
		ShowHiddenModels:   (r.ShowHiddenModels == "true"),
		ShowHiddenExplores: (r.ShowHiddenExplores == "true"),
		ShowHiddenFields:   (r.ShowHiddenFields == "true"),
	}
	return s, nil

}

var _ sources.Source = &Source{}

type Source struct {
	Name               string `yaml:"name"`
	Kind               string `yaml:"kind"`
	Timeout            string `yaml:"timeout"`
	Client             *v4.LookerSDK
	ApiSettings        *rtl.ApiSettings
	ShowHiddenModels   bool `yaml:"show_hidden_models"`
	ShowHiddenExplores bool `yaml:"show_hidden_explores"`
	ShowHiddenFields   bool `yaml:"show_hidden_fields"`
}

func (s *Source) SourceKind() string {
	return SourceKind
}
