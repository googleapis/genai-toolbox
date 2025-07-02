// Copyright 2024 Google LLC
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

package mongodb_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/mongodb"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlMongoDB(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				mongo-db:
					kind: "mongodb"
					uri: "mongodb+srv://username:password@host/dbname"
			`,
			want: server.SourceConfigs{
				"mongo-db": mongodb.Config{
					Name: "mongo-db",
					Kind: mongodb.SourceKind,
					Uri:  "mongodb+srv://username:password@host/dbname",
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

func TestFailParseFromYaml(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "extra field",
			in: `
			sources:
				mongo-db:
					kind: mongodb
					uri: "mongodb+srv://username:password@host/dbname"
					foo: bar
			`,
			err: "unable to parse source \"mongo-db\" as \"mongodb\": [1:1] unknown field \"foo\"\n>  1 | foo: bar\n       ^\n   2 | kind: mongodb\n   3 | uri: mongodb+srv://username:password@host/dbname",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				mongo-db:
					kind: mongodb
			`,
			err: "unable to parse source \"mongo-db\" as \"mongodb\": Key: 'Config.Uri' Error:Field validation for 'Uri' failed on the 'required' tag",
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
				t.Fatalf("unexpected error: got \n%q, want \n%q", errStr, tc.err)
			}
		})
	}
}
