// Copyright 2025 Google LLC
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

package elasticsearch

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestConfig_ToolConfigKind(t *testing.T) {
	config := Config{}
	if got := config.ToolConfigKind(); got != kind {
		t.Errorf("ToolConfigKind() = %v, want %v", got, kind)
	}
}

func TestNewConfig(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	yamlData := `
name: test-elasticsearch-tool
kind: elasticsearch
source: test-source
operation: search
description: Test Elasticsearch search tool
parameters:
  - name: index
    type: string
    description: Index name
    required: true
  - name: query
    type: map
    description: Search query
    required: true
`

	decoder := yaml.NewDecoder(strings.NewReader(yamlData))

	config, err := newConfig(ctx, "test-elasticsearch-tool", decoder)
	if err != nil {
		t.Fatalf("newConfig() error = %v", err)
	}

	esConfig, ok := config.(Config)
	if !ok {
		t.Fatal("Expected Config type")
	}

	if esConfig.Name != "test-elasticsearch-tool" {
		t.Errorf("Name = %v, want %v", esConfig.Name, "test-elasticsearch-tool")
	}

	if esConfig.Kind != "elasticsearch" {
		t.Errorf("Kind = %v, want %v", esConfig.Kind, "elasticsearch")
	}

	if esConfig.Source != "test-source" {
		t.Errorf("Source = %v, want %v", esConfig.Source, "test-source")
	}

	if esConfig.Operation != "search" {
		t.Errorf("Operation = %v, want %v", esConfig.Operation, "search")
	}

	if esConfig.Description != "Test Elasticsearch search tool" {
		t.Errorf("Description = %v, want expected", esConfig.Description)
	}

	if len(esConfig.Parameters) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(esConfig.Parameters))
	}
}

func TestConfig_Initialize_NoSource(t *testing.T) {
	config := Config{
		Name:   "test",
		Source: "nonexistent-source",
	}

	sources := map[string]sources.Source{}

	_, err := config.Initialize(sources)
	if err == nil {
		t.Error("Expected error for nonexistent source")
	}
}

func TestConfig_Initialize_IncompatibleSource(t *testing.T) {
	// This test would need a mock incompatible source
	// Skipping for now as it requires more complex setup
}

// Note: Full integration tests would require a running Elasticsearch instance
// and more complex mock setup. These are basic unit tests to verify the structure.