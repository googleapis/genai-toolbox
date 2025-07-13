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

package mssql_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/mssql"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlMssql(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-mssql-instance:
					kind: mssql
					host: 0.0.0.0
					port: my-port
					database: my_db
					user: my_user
					password: my_pass
			`,
			want: server.SourceConfigs{
				"my-mssql-instance": mssql.Config{
					Name:     "my-mssql-instance",
					Kind:     mssql.SourceKind,
					Host:     "0.0.0.0",
					Port:     "my-port",
					Database: "my_db",
					User:     "my_user",
					Password: "my_pass",
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
				t.Fatalf("incorrect psarse: want %v, got %v", tc.want, got.Sources)
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
				my-mssql-instance:
					kind: mssql
					host: 0.0.0.0
					port: my-port
					database: my_db
					user: my_user
					password: my_pass
					foo: bar
			`,
			err: "unable to parse source \"my-mssql-instance\" as \"mssql\": [2:1] unknown field \"foo\"\n   1 | database: my_db\n>  2 | foo: bar\n       ^\n   3 | host: 0.0.0.0\n   4 | kind: mssql\n   5 | password: my_pass\n   6 | ",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-mssql-instance:
					kind: mssql
					host: 0.0.0.0
					port: my-port
					database: my_db
					user: my_user
			`,
			err: "unable to parse source \"my-mssql-instance\" as \"mssql\": Key: 'Config.Password' Error:Field validation for 'Password' failed on the 'required' tag",
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

// TestParseEncryptField checks that the encrypt field is parsed correctly from YAML.
func TestParseEncryptField(t *testing.T) {
	tests := []struct {
		desc    string
		encrypt string
		want    string
	}{
		{"empty encrypt", "", ""},
		{"encrypt true", "true", "true"},
		{"encrypt disable", "disable", "disable"},
		{"encrypt strict", "strict", "strict"},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			yamlInput := `
sources:
  my-mssql-instance:
    kind: mssql
    host: 0.0.0.0
    port: my-port
    database: my_db
    user: my_user
    password: my_pass
    encrypt: ` + tc.encrypt + `
`
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			err := yaml.Unmarshal(testutils.FormatYaml(yamlInput), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			cfg, ok := got.Sources["my-mssql-instance"].(mssql.Config)
			if !ok {
				t.Fatalf("unexpected config type")
			}
			if cfg.Encrypt != tc.want {
				t.Errorf("Encrypt: got %q, want %q", cfg.Encrypt, tc.want)
			}
		})
	}
}

// TestMissingEncryptField checks that the encrypt field is optional and defaults to empty.
func TestMissingEncryptField(t *testing.T) {
	yamlInput := `
sources:
  my-mssql-instance:
    kind: mssql
    host: 0.0.0.0
    port: my-port
    database: my_db
    user: my_user
    password: my_pass
`
	got := struct {
		Sources server.SourceConfigs `yaml:"sources"`
	}{}
	err := yaml.Unmarshal(testutils.FormatYaml(yamlInput), &got)
	if err != nil {
		t.Fatalf("unable to unmarshal: %s", err)
	}
	cfg, ok := got.Sources["my-mssql-instance"].(mssql.Config)
	if !ok {
		t.Fatalf("unexpected config type")
	}
	if cfg.Encrypt != "" {
		t.Errorf("Encrypt: got %q, want empty string", cfg.Encrypt)
	}
}
