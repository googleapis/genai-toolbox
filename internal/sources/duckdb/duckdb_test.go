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

package duckdb_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/duckdb"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParserFromYamlDuckDb(t *testing.T) {
	config := make(map[string]string)
	config["access_mode"] = "READ_ONLY"
	config["threads"] = "4"
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-duckdb:
					kind: duckdb
			`,
			want: server.SourceConfigs{
				"my-duckdb": duckdb.Config{
					Name: "my-duckdb",
					Kind: duckdb.SourceKind,
				},
			},
		},
		{
			desc: "with custom configuration",
			in: `
			sources:
				my-duckdb:
					kind: duckdb
					configuration: 
						access_mode: READ_ONLY
						threads: 4
			`,
			want: server.SourceConfigs{
				"my-duckdb": duckdb.Config{
					Name:          "my-duckdb",
					Kind:          duckdb.SourceKind,
					Configuration: config,
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
