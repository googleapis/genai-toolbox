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
package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "http"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{Name: name, Timeout: "30s"} // Default timeout
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name                   string            `yaml:"name" validate:"required"`
	Kind                   string            `yaml:"kind" validate:"required"`
	BaseURL                string            `yaml:"baseUrl"`
	Timeout                string            `yaml:"timeout"`
	DefaultHeaders         map[string]string `yaml:"headers"`
	QueryParams            map[string]string `yaml:"queryParams"`
	DisableSslVerification bool              `yaml:"disableSslVerification"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

// Initialize initializes an HTTP Source instance.
func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	duration, err := time.ParseDuration(r.Timeout)
	if err != nil {
		return nil, fmt.Errorf("unable to parse Timeout string as time.Duration: %s", err)
	}

	tr := &http.Transport{}

	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}

	if r.DisableSslVerification {
		tr.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}

		logger.WarnContext(ctx, "Insecure HTTP is enabled for HTTP source %s. TLS certificate verification is skipped.\n", r.Name)
	}

	client := http.Client{
		Timeout:   duration,
		Transport: tr,
	}

	// Validate BaseURL
	_, err = url.ParseRequestURI(r.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse BaseUrl %v", err)
	}

	ua, err := util.UserAgentFromContext(ctx)
	if err != nil {
		fmt.Printf("Error in User Agent retrieval: %s", err)
	}
	if r.DefaultHeaders == nil {
		r.DefaultHeaders = make(map[string]string)
	}
	if existingUA, ok := r.DefaultHeaders["User-Agent"]; ok {
		ua = ua + " " + existingUA
	}
	r.DefaultHeaders["User-Agent"] = ua

	s := &Source{
		Config: r,
		Client: &client,
	}
	return s, nil

}

var _ sources.Source = &Source{}

type Source struct {
	Config
	Client *http.Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}
