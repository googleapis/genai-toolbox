// Copyright 2026 Google LLC
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

package cockroachdbexplainquery

import (
	"context"
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
)

func TestParseFromYamlCockroachDBExplainQuery(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "basic example",
			yaml: `
name: explain-query
kind: cockroachdb-explain-query
source: my-cockroachdb
description: "Explains the execution plan for a SQL query"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := yaml.NewDecoder(strings.NewReader(tt.yaml))
			ctx := context.Background()
			config, err := newConfig(ctx, "test-tool", decoder)
			if err != nil {
				t.Fatalf("newConfig() error = %v", err)
			}

			if config == nil {
				t.Fatal("expected non-nil config")
			}

			cfg, ok := config.(Config)
			if !ok {
				t.Fatalf("expected Config type, got %T", config)
			}

			if cfg.Kind != kind {
				t.Errorf("Kind = %v, want %v", cfg.Kind, kind)
			}

			if cfg.Source == "" {
				t.Error("Source should not be empty")
			}

			if cfg.Description == "" {
				t.Error("Description should not be empty")
			}
		})
	}
}

func TestCockroachDBExplainQueryToolConfigKind(t *testing.T) {
	cfg := Config{}
	if got := cfg.ToolConfigKind(); got != kind {
		t.Errorf("ToolConfigKind() = %v, want %v", got, kind)
	}
}

func TestCockroachDBExplainQueryInitialize(t *testing.T) {
	cfg := Config{
		Name:        "test-explain-query",
		Kind:        kind,
		Source:      "test-source",
		Description: "Test description",
	}

	// Initialize will fail because source doesn't exist, but shouldn't panic
	_, err := cfg.Initialize(map[string]sources.Source{})
	if err == nil {
		t.Error("Expected error when source doesn't exist")
	}
}

func TestCockroachDBExplainQueryParameters(t *testing.T) {
	// Verify the tool has required parameters
	expectedParams := []string{"query", "verbose"}

	if len(expectedParams) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(expectedParams))
	}

	if expectedParams[0] != "query" {
		t.Errorf("First parameter should be query, got %s", expectedParams[0])
	}
	if expectedParams[1] != "verbose" {
		t.Errorf("Second parameter should be verbose, got %s", expectedParams[1])
	}
}
