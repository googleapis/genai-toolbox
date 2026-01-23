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

package firestore_test

import (
	"testing"
	"time"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/firestore"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlFirestore(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example with default database",
			in: `
			sources:
				my-firestore:
					kind: firestore
					project: my-project
			`,
			want: server.SourceConfigs{
				"my-firestore": firestore.Config{
					Name:     "my-firestore",
					Kind:     firestore.SourceKind,
					Project:  "my-project",
					Database: "",
				},
			},
		},
		{
			desc: "with custom database",
			in: `
			sources:
				my-firestore:
					kind: firestore
					project: my-project
					database: my-database
			`,
			want: server.SourceConfigs{
				"my-firestore": firestore.Config{
					Name:     "my-firestore",
					Kind:     firestore.SourceKind,
					Project:  "my-project",
					Database: "my-database",
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if !cmp.Equal(tc.want, got.Sources) {
				t.Fatalf("incorrect parse: want %v, got %v", tc.want, got.Sources)
			}
		})
	}
}

func TestFailParseFromYamlFirestore(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "extra field",
			in: `
			sources:
				my-firestore:
					kind: firestore
					project: my-project
					foo: bar
			`,
			err: "unable to parse source \"my-firestore\" as \"firestore\": [1:1] unknown field \"foo\"\n>  1 | foo: bar\n       ^\n   2 | kind: firestore\n   3 | project: my-project",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-firestore:
					kind: firestore
					database: my-database
			`,
			err: "unable to parse source \"my-firestore\" as \"firestore\": Key: 'Config.Project' Error:Field validation for 'Project' failed on the 'required' tag",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			errStr := err.Error()
			if errStr != tc.err {
				t.Fatalf("unexpected error: got %q, want %q", errStr, tc.err)
			}
		})
	}
}

func TestFirestoreValueToJSON_RoundTrip(t *testing.T) {
	// Test round-trip conversion
	original := map[string]any{
		"name":   "Test",
		"count":  int64(42),
		"price":  19.99,
		"active": true,
		"tags":   []any{"tag1", "tag2"},
		"metadata": map[string]any{
			"created": time.Now(),
		},
		"nullField": nil,
	}

	// Convert to JSON representation
	jsonRepresentation := firestore.FirestoreValueToJSON(original)

	// Verify types are simplified
	jsonMap, ok := jsonRepresentation.(map[string]any)
	if !ok {
		t.Fatalf("Expected map, got %T", jsonRepresentation)
	}

	// Time should be converted to string
	metadata, ok := jsonMap["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata should be a map, got %T", jsonMap["metadata"])
	}
	_, ok = metadata["created"].(string)
	if !ok {
		t.Errorf("created should be a string, got %T", metadata["created"])
	}
}
