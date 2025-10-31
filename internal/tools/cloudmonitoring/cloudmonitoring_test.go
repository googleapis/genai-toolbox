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
	"github.com/googleapis/genai-toolbox/internal/sources"
	cloudmonitoringsrc "github.com/googleapis/genai-toolbox/internal/sources/cloudmonitoring"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestInitialize(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Name:        "test-tool",
		Kind:        kind,
		Source:      "test-source",
		Description: "A test tool",
	}

	srcs := map[string]sources.Source{
		"test-source": &cloudmonitoringsrc.Source{
			BaseURL: "http://localhost",
			Client:  &http.Client{},
		},
	}

	itool, err := cfg.Initialize(srcs)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	tool, ok := itool.(Tool)
	if !ok {
		t.Fatalf("Initialize() did not return a cloudmonitoring.Tool")
	}

	if tool.Name != "test-tool" {
		t.Errorf("tool.Name = %q, want %q", tool.Name, "test-tool")
	}
}

func TestInitialize_NoSource(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Name:   "test-tool",
		Kind:   kind,
		Source: "test-source",
	}

	srcs := map[string]sources.Source{}

	_, err := cfg.Initialize(srcs)
	if err == nil {
		t.Fatal("Initialize() error = nil, want error")
	}
}

func TestInitialize_InvalidSource(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Name:   "test-tool",
		Kind:   kind,
		Source: "test-source",
	}

	// A dummy source that is not a cloudmonitoring source
	type invalidSource struct{ sources.Source }
	srcs := map[string]sources.Source{
		"test-source": &invalidSource{},
	}

	_, err := cfg.Initialize(srcs)
	if err == nil {
		t.Fatal("Initialize() error = nil, want error")
	}
}

func TestToolConfigKind(t *testing.T) {
	t.Parallel()
	cfg := Config{}
	if cfg.ToolConfigKind() != kind {
		t.Errorf("ToolConfigKind() = %q, want %q", cfg.ToolConfigKind(), kind)
	}
}

func TestNewConfig(t *testing.T) {
	t.Parallel()
	yamlString := `
name: test-tool
kind: cloud-monitoring-query-prometheus
source: test-source
description: A test tool
`
	decoder := yaml.NewDecoder(strings.NewReader(yamlString))
	cfg, err := newConfig(context.Background(), "test-tool", decoder)
	if err != nil {
		t.Fatalf("newConfig() error = %v", err)
	}

	expected := Config{
		Name:        "test-tool",
		Kind:        "cloud-monitoring-query-prometheus",
		Source:      "test-source",
		Description: "A test tool",
	}

	if diff := cmp.Diff(expected, cfg); diff != "" {
		t.Errorf("newConfig() mismatch (-want +got): %s", diff)
	}
}

func TestParseParams(t *testing.T) {
	t.Parallel()
	tool := Tool{
		AllParams: tools.Parameters{
			tools.NewStringParameterWithRequired("projectId", "The Id of the Google Cloud project.", true),
			tools.NewStringParameterWithRequired("query", "The promql query to execute.", true),
		},
	}

	data := map[string]any{
		"projectId": "test-project",
		"query":     "up",
	}

	params, err := tool.ParseParams(data, nil)
	if err != nil {
		t.Fatalf("ParseParams() error = %v", err)
	}

	expected := tools.ParamValues{
		{Name: "projectId", Value: "test-project"},
		{Name: "query", Value: "up"},
	}

	if diff := cmp.Diff(expected, params); diff != "" {
		t.Errorf("ParseParams() mismatch (-want +got): %s", diff)
	}
}

func TestManifest(t *testing.T) {
	t.Parallel()
	expected := tools.Manifest{Description: "desc"}
	tool := Tool{manifest: expected}
	if diff := cmp.Diff(expected, tool.Manifest()); diff != "" {
		t.Errorf("Manifest() mismatch (-want +got): %s", diff)
	}
}

func TestMcpManifest(t *testing.T) {
	t.Parallel()
	expected := tools.McpManifest{Name: "mcp-manifest"}
	tool := Tool{mcpManifest: expected}
	if diff := cmp.Diff(expected, tool.McpManifest()); diff != "" {
		t.Errorf("McpManifest() mismatch (-want +got): %s", diff)
	}
}

func TestAuthorized(t *testing.T) {
	t.Parallel()
	tool := Tool{}
	if !tool.Authorized(nil) {
		t.Error("Authorized() = false, want true")
	}
}

func TestRequiresClientAuthorization(t *testing.T) {
	t.Parallel()
	tool := Tool{}
	if tool.RequiresClientAuthorization() {
		t.Error("RequiresClientAuthorization() = true, want false")
	}
}
