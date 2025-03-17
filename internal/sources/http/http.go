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
	"net/http"
	"time"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "http"

// validate interface
var _ sources.SourceConfig = Config{}

type Config struct {
	Name        string            `yaml:"name" validate:"required"`
	Kind        string            `yaml:"kind" validate:"required"`
	BaseURL     string            `yaml:"baseUrl"`
	Timeout     int               `yaml:"timeout"`
	Headers     map[string]string `yaml:"headers"`
	QueryParams map[string]string `yaml:"queryParams"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client := http.Client{
		Timeout: 30 * time.Second,
	}

	s := &Source{
		Name:        r.Name,
		Kind:        SourceKind,
		BaseURL:     r.BaseURL,
		Timeout:     r.Timeout,
		Headers:     r.Headers,
		QueryParams: r.QueryParams,
		Client:      &client,
	}
	return s, nil

}

var _ sources.Source = &Source{}

type Source struct {
	Name        string            `yaml:"name"`
	Kind        string            `yaml:"kind"`
	BaseURL     string            `yaml:"baseUrl"`
	Timeout     int               `yaml:"timeout"`
	Headers     map[string]string `yaml:"headers"`
	QueryParams map[string]string `yaml:"queryParams"`
	Client      *http.Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) HTTPClient() *http.Client {
	return s.Client
}

func (s *Source) GetBaseURL() string {
	return s.BaseURL
}

func (s *Source) GetHeaders() map[string]string {
	return s.Headers
}

func (s *Source) GetQueryParams() map[string]string {
	return s.Headers
}
