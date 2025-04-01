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
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "http"

// validate interface
var _ sources.SourceConfig = Config{}

type Config struct {
	Name           string            `yaml:"name" validate:"required"`
	Kind           string            `yaml:"kind" validate:"required"`
	BaseURL        string            `yaml:"baseUrl"`
	Timeout        string            `yaml:"timeout"`
	DefaultHeaders map[string]string `yaml:"headers"`
	QueryParams    map[string]string `yaml:"queryParams"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

// DefaultConfig is a helper function that generates the default configuration for an HTTP Tool Config.
func DefaultConfig(name string) Config {
	return Config{Name: name, Timeout: "30s"}
}

// Initialize initializes an HTTP Source instance.
func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	duration, err := time.ParseDuration(r.Timeout)
	if err != nil {
		return nil, fmt.Errorf("unable to parse Timeout string as time.Duration: %s", err)
	}
	client := http.Client{
		Timeout: duration,
	}

	// Validate BaseURL
	_, err = url.ParseRequestURI(r.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse BaseUrl %v", err)
	}

	s := &Source{
		Name:           r.Name,
		Kind:           SourceKind,
		BaseURL:        r.BaseURL,
		DefaultHeaders: r.DefaultHeaders,
		QueryParams:    r.QueryParams,
		Client:         &client,
	}
	return s, nil

}

var _ sources.Source = &Source{}

type Source struct {
	Name           string            `yaml:"name"`
	Kind           string            `yaml:"kind"`
	BaseURL        string            `yaml:"baseUrl"`
	DefaultHeaders map[string]string `yaml:"headers"`
	QueryParams    map[string]string `yaml:"queryParams"`
	Client         *http.Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}
