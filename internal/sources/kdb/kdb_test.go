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

package kdb_test

import (
	"context"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/kdb"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlKDB(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-kdb-instance:
					kind: kdb
					host: my-host
					port: 5001
					username: my_user
					password: my_pass
			`,
			want: server.SourceConfigs{
				"my-kdb-instance": &kdb.Config{
					Name:     "my-kdb-instance",
					Kind:     kdb.SourceKind,
					Host:     "my-host",
					Port:     5001,
					Username: "my_user",
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
					kind: kdb
					host: my-host
					port: 5001
					username: my_user
					password: my_pass
					foo: bar
			`,
			err: "unknown field",
		},
		{
			desc: "missing required field",
			in: `
					kind: kdb
					port: 5001
					username: my_user
					password: my_pass
			`,
			err: "Error:Field validation for 'Host' failed on the 'required' tag",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			var v map[string]any
			if err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &v); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			b, err := yaml.Marshal(v)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			decoder := yaml.NewDecoder(
				strings.NewReader(string(b)),
				yaml.Strict(),
				yaml.Validator(validator.New()),
			)

			_, err = sources.DecodeConfig(context.Background(), kdb.SourceKind, "my-kdb-instance", decoder)
			if err == nil {
				t.Fatalf("expected unmarshal to fail")
			}
			if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("incorrect error: got %q, want %q", err.Error(), tc.err)
			}
		})
	}
}

