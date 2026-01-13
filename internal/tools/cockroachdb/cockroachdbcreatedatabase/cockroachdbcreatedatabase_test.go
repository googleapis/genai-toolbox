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

package cockroachdbcreatedatabase

import (
	"context"
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
)

func TestParseFromYamlCockroachDBCreateDatabase(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "basic example",
			yaml: `
name: create-database
kind: cockroachdb-create-database
source: my-cockroachdb
description: "Creates a new database (requires write mode)"
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
		})
	}
}

func TestCockroachDBCreateDatabaseToolConfigKind(t *testing.T) {
	cfg := Config{}
	if got := cfg.ToolConfigKind(); got != kind {
		t.Errorf("ToolConfigKind() = %v, want %v", got, kind)
	}
}

func TestCockroachDBCreateDatabaseInitialize(t *testing.T) {
	cfg := Config{
		Name:        "test-create-database",
		Kind:        kind,
		Source:      "test-source",
		Description: "Test description",
	}

	_, err := cfg.Initialize(map[string]sources.Source{})
	if err == nil {
		t.Error("Expected error when source doesn't exist")
	}
}

func TestCockroachDBCreateDatabaseParameters(t *testing.T) {
	expectedParams := []string{"database_name"}

	if len(expectedParams) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(expectedParams))
	}

	if expectedParams[0] != "database_name" {
		t.Errorf("Parameter should be database_name, got %s", expectedParams[0])
	}
}
