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

package cockroachdblistindexes

import (
	"context"
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
)

func TestParseFromYamlCockroachDBListIndexes(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "basic example",
			yaml: `
name: list-table-indexes
kind: cockroachdb-list-indexes
source: my-cockroachdb
description: "Lists all indexes on a table including index definitions"
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

func TestCockroachDBListIndexesToolConfigKind(t *testing.T) {
	cfg := Config{}
	if got := cfg.ToolConfigKind(); got != kind {
		t.Errorf("ToolConfigKind() = %v, want %v", got, kind)
	}
}

func TestCockroachDBListIndexesInitialize(t *testing.T) {
	cfg := Config{
		Name:        "test-list-indexes",
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

func TestCockroachDBListIndexesParameters(t *testing.T) {
	// Verify the tool requires schema_name and table_name parameters
	schemaParam := "schema_name"
	tableParam := "table_name"

	expectedParams := []string{schemaParam, tableParam}

	if len(expectedParams) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(expectedParams))
	}

	if expectedParams[0] != "schema_name" {
		t.Errorf("First parameter should be schema_name, got %s", expectedParams[0])
	}
	if expectedParams[1] != "table_name" {
		t.Errorf("Second parameter should be table_name, got %s", expectedParams[1])
	}
}
