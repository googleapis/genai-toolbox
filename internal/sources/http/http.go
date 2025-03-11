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
	"time"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opencensus.io/trace"
)

const SourceKind string = "http"

// validate interface
var _ sources.SourceConfig = Config{}

type Config struct {
	Name        string              `yaml:"name" validate:"required"`
	Kind        string              `yaml:"kind" validate:"required"`
	BaseURL     string              `yaml:"baseUrl"`
	timeout     int                 `yaml:"timeout"`
	headers     []map[string]string `yaml:"headers"`
	queryParams []map[string]string `yaml:"queryParams"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client := http.Client{
		Timeout: 30 * time.Second,
	}

	fmt.Printf("client: status code: %d", res.StatusCode)
	s := &Source{
		Name: r.Name,
		Kind: SourceKind,
		Pool: pool,
	}
	return s, nil

}

var _ sources.Source = &Source{}

type Source struct {
	Name        string `yaml:"name"`
	Kind        string `yaml:"kind"`
	Client      *http.Client
	headers     []map[string]string
	queryParams []map[string]string
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) HTTPClient() *http.Client {
	return s.Client
}
