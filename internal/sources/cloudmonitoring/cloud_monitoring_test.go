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

package cloudmonitoring

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
)

func TestInitialize(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Name: "test-source",
		Kind: SourceKind,
	}

	ctx := util.WithUserAgent(context.Background(), "test-agent")

	source, err := cfg.Initialize(ctx, trace.NewNoopTracerProvider().Tracer(""))
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	if source.SourceKind() != SourceKind {
		t.Errorf("source.SourceKind() = %q, want %q", source.SourceKind(), SourceKind)
	}
}

func TestInitialize_WithClientOAuth(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Name:           "test-source",
		Kind:           SourceKind,
		UseClientOAuth: true,
	}

	ctx := util.WithUserAgent(context.Background(), "test-agent")

	isource, err := cfg.Initialize(ctx, trace.NewNoopTracerProvider().Tracer(""))
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	source, ok := isource.(*Source)
	if !ok {
		t.Fatalf("Initialize() did not return a *Source")
	}

	if !source.UseClientAuthorization() {
		t.Error("source.UseClientAuthorization() = false, want true")
	}
}

func TestNewConfig(t *testing.T) {
	t.Parallel()
	yamlString := `
name: test-source
kind: cloud-monitoring
`
	decoder := yaml.NewDecoder(strings.NewReader(yamlString))
	cfg, err := newConfig(context.Background(), "test-source", decoder)
	if err != nil {
		t.Fatalf("newConfig() error = %v", err)
	}

	expected := Config{
		Name: "test-source",
		Kind: "cloud-monitoring",
	}

	if diff := cmp.Diff(expected, cfg); diff != "" {
		t.Errorf("newConfig() mismatch (-want +got): %s", diff)
	}
}

func TestSourceConfigKind(t *testing.T) {
	t.Parallel()
	cfg := Config{}
	if cfg.SourceConfigKind() != SourceKind {
		t.Errorf("SourceConfigKind() = %q, want %q", cfg.SourceConfigKind(), SourceKind)
	}
}

func TestSourceKind(t *testing.T) {
	t.Parallel()
	source := Source{}
	if source.SourceKind() != SourceKind {
		t.Errorf("SourceKind() = %q, want %q", source.SourceKind(), SourceKind)
	}
}

func TestGetClient(t *testing.T) {
	t.Parallel()
	source := &Source{Client: &http.Client{}}

	client, err := source.GetClient(context.Background(), "")
	if err != nil {
		t.Fatalf("GetClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("GetClient() client is nil")
	}
}

func TestGetClient_WithClientOAuth(t *testing.T) {
	t.Parallel()
	source := &Source{UseClientOAuth: true}

	client, err := source.GetClient(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("GetClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("GetClient() client is nil")
	}
}

func TestGetClient_WithClientOAuth_NoToken(t *testing.T) {
	t.Parallel()
	source := &Source{UseClientOAuth: true}

	_, err := source.GetClient(context.Background(), "")
	if err == nil {
		t.Fatal("GetClient() error = nil, want error")
	}
}

func TestUseClientAuthorization(t *testing.T) {
	t.Parallel()
	source := &Source{UseClientOAuth: true}
	if !source.UseClientAuthorization() {
		t.Error("UseClientAuthorization() = false, want true")
	}

	source = &Source{UseClientOAuth: false}
	if source.UseClientAuthorization() {
		t.Error("UseClientAuthorization() = true, want false")
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	mockRT := &mockRoundTripper{}
	roundTripper := &userAgentRoundTripper{
		userAgent: "test-agent",
		next:      mockRT,
	}

	req, err := http.NewRequest("GET", "http://localhost", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	_, err = roundTripper.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}

	if mockRT.req.Header.Get("User-Agent") != "test-agent" {
		t.Errorf("User-Agent header = %q, want %q", mockRT.req.Header.Get("User-Agent"), "test-agent")
	}
}

func TestRoundTrip_WithExistingUserAgent(t *testing.T) {
	t.Parallel()

	mockRT := &mockRoundTripper{}
	roundTripper := &userAgentRoundTripper{
		userAgent: "test-agent",
		next:      mockRT,
	}

	req, err := http.NewRequest("GET", "http://localhost", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	req.Header.Set("User-Agent", "existing-agent")

	_, err = roundTripper.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}

	if mockRT.req.Header.Get("User-Agent") != "existing-agent test-agent" {
		t.Errorf("User-Agent header = %q, want %q", mockRT.req.Header.Get("User-Agent"), "existing-agent test-agent")
	}
}

type mockRoundTripper struct {
	req *http.Request
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.req = req
	return &http.Response{StatusCode: http.StatusOK}, nil
}