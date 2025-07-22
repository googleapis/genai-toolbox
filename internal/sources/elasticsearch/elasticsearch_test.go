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
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestConfig_SourceConfigKind(t *testing.T) {
	config := Config{}
	if got := config.SourceConfigKind(); got != SourceKind {
		t.Errorf("SourceConfigKind() = %v, want %v", got, SourceKind)
	}
}

func TestNewConfig(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	yamlData := `
name: test-elasticsearch
kind: elasticsearch
addresses:
  - http://localhost:9200
username: elastic
password: changeme
`

	decoder := yaml.NewDecoder(strings.NewReader(yamlData))

	config, err := newConfig(ctx, "test-elasticsearch", decoder)
	if err != nil {
		t.Fatalf("newConfig() error = %v", err)
	}

	elasticsearchConfig, ok := config.(Config)
	if !ok {
		t.Fatal("Expected Config type")
	}

	if elasticsearchConfig.Name != "test-elasticsearch" {
		t.Errorf("Name = %v, want %v", elasticsearchConfig.Name, "test-elasticsearch")
	}

	if elasticsearchConfig.Kind != "elasticsearch" {
		t.Errorf("Kind = %v, want %v", elasticsearchConfig.Kind, "elasticsearch")
	}

	if len(elasticsearchConfig.Addresses) != 1 || elasticsearchConfig.Addresses[0] != "http://localhost:9200" {
		t.Errorf("Addresses = %v, want [http://localhost:9200]", elasticsearchConfig.Addresses)
	}

	if elasticsearchConfig.Username != "elastic" {
		t.Errorf("Username = %v, want %v", elasticsearchConfig.Username, "elastic")
	}

	if elasticsearchConfig.Password != "changeme" {
		t.Errorf("Password = %v, want %v", elasticsearchConfig.Password, "changeme")
	}
}

func TestSource_SourceKind(t *testing.T) {
	source := &Source{}
	if got := source.SourceKind(); got != SourceKind {
		t.Errorf("SourceKind() = %v, want %v", got, SourceKind)
	}
}

// Note: Full integration tests would require a running Elasticsearch instance
// This is a basic unit test to verify the structure and interfaces