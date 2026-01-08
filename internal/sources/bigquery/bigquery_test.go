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

package bigquery_test

import (
	"math/big"
	"reflect"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/bigquery"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/util"
)

func TestParseFromYamlBigQuery(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-instance:
					kind: bigquery
					project: my-project
			`,
			want: server.SourceConfigs{
				"my-instance": bigquery.Config{
					Name:      "my-instance",
					Kind:      bigquery.SourceKind,
					Project:   "my-project",
					Location:  "",
					WriteMode: "",
				},
			},
		},
		{
			desc: "all fields specified",
			in: `
			sources:
				my-instance:
					kind: bigquery
					project: my-project
					location: asia
					writeMode: blocked
			`,
			want: server.SourceConfigs{
				"my-instance": bigquery.Config{
					Name:           "my-instance",
					Kind:           bigquery.SourceKind,
					Project:        "my-project",
					Location:       "asia",
					WriteMode:      "blocked",
					UseClientOAuth: false,
				},
			},
		},
		{
			desc: "use client auth example",
			in: `
			sources:
				my-instance:
					kind: bigquery
					project: my-project
					location: us
					useClientOAuth: true
			`,
			want: server.SourceConfigs{
				"my-instance": bigquery.Config{
					Name:           "my-instance",
					Kind:           bigquery.SourceKind,
					Project:        "my-project",
					Location:       "us",
					UseClientOAuth: true,
				},
			},
		},
		{
			desc: "with allowed datasets example",
			in: `
			sources:
				my-instance:
					kind: bigquery
					project: my-project
					location: us
					allowedDatasets:
						- my_dataset
			`,
			want: server.SourceConfigs{
				"my-instance": bigquery.Config{
					Name:            "my-instance",
					Kind:            bigquery.SourceKind,
					Project:         "my-project",
					Location:        "us",
					AllowedDatasets: []string{"my_dataset"},
				},
			},
		},
		{
			desc: "with service account impersonation example",
			in: `
			sources:
				my-instance:
					kind: bigquery
					project: my-project
					location: us
					impersonateServiceAccount: service-account@my-project.iam.gserviceaccount.com
			`,
			want: server.SourceConfigs{
				"my-instance": bigquery.Config{
					Name:                      "my-instance",
					Kind:                      bigquery.SourceKind,
					Project:                   "my-project",
					Location:                  "us",
					ImpersonateServiceAccount: "service-account@my-project.iam.gserviceaccount.com",
				},
			},
		},
		{
			desc: "with custom scopes example",
			in: `
			sources:
				my-instance:
					kind: bigquery
					project: my-project
					location: us
					scopes:
						- https://www.googleapis.com/auth/bigquery
						- https://www.googleapis.com/auth/cloud-platform
			`,
			want: server.SourceConfigs{
				"my-instance": bigquery.Config{
					Name:     "my-instance",
					Kind:     bigquery.SourceKind,
					Project:  "my-project",
					Location: "us",
					Scopes:   []string{"https://www.googleapis.com/auth/bigquery", "https://www.googleapis.com/auth/cloud-platform"},
				},
			},
		},
		{
			desc: "with max query result rows example",
			in: `
			sources:
				my-instance:
					kind: bigquery
					project: my-project
					location: us
					maxQueryResultRows: 10
			`,
			want: server.SourceConfigs{
				"my-instance": bigquery.Config{
					Name:               "my-instance",
					Kind:               bigquery.SourceKind,
					Project:            "my-project",
					Location:           "us",
					MaxQueryResultRows: 10,
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
				my-instance:
					kind: bigquery
					project: my-project
					location: us
					foo: bar
			`,
			err: "unable to parse source \"my-instance\" as \"bigquery\": [1:1] unknown field \"foo\"\n>  1 | foo: bar\n       ^\n   2 | kind: bigquery\n   3 | location: us\n   4 | project: my-project",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-instance:
					kind: bigquery
					location: us
			`,
			err: "unable to parse source \"my-instance\" as \"bigquery\": Key: 'Config.Project' Error:Field validation for 'Project' failed on the 'required' tag",
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

func TestInitialize_MaxQueryResultRows(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	ctx = util.WithUserAgent(ctx, "test-agent")
	tracer := noop.NewTracerProvider().Tracer("")

	tcs := []struct {
		desc string
		cfg  bigquery.Config
		want int
	}{
		{
			desc: "default value",
			cfg: bigquery.Config{
				Name:           "test-default",
				Kind:           bigquery.SourceKind,
				Project:        "test-project",
				UseClientOAuth: true,
			},
			want: 50,
		},
		{
			desc: "configured value",
			cfg: bigquery.Config{
				Name:               "test-configured",
				Kind:               bigquery.SourceKind,
				Project:            "test-project",
				UseClientOAuth:     true,
				MaxQueryResultRows: 100,
			},
			want: 100,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			src, err := tc.cfg.Initialize(ctx, tracer)
			if err != nil {
				t.Fatalf("Initialize failed: %v", err)
			}
			bqSrc, ok := src.(*bigquery.Source)
			if !ok {
				t.Fatalf("Expected *bigquery.Source, got %T", src)
			}
			if bqSrc.MaxQueryResultRows != tc.want {
				t.Errorf("MaxQueryResultRows = %d, want %d", bqSrc.MaxQueryResultRows, tc.want)
			}
		})
	}
}

func TestNormalizeValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{
			name:     "big.Rat 1/3 (NUMERIC scale 9)",
			input:    new(big.Rat).SetFrac64(1, 3),               // 0.33333333333...
			expected: "0.33333333333333333333333333333333333333", // FloatString(38)
		},
		{
			name:     "big.Rat 19/2 (9.5)",
			input:    new(big.Rat).SetFrac64(19, 2),
			expected: "9.5",
		},
		{
			name:     "big.Rat 12341/10 (1234.1)",
			input:    new(big.Rat).SetFrac64(12341, 10),
			expected: "1234.1",
		},
		{
			name:     "big.Rat 10/1 (10)",
			input:    new(big.Rat).SetFrac64(10, 1),
			expected: "10",
		},
		{
			name:     "string",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "int",
			input:    123,
			expected: 123,
		},
		{
			name: "nested slice of big.Rat",
			input: []any{
				new(big.Rat).SetFrac64(19, 2),
				new(big.Rat).SetFrac64(1, 4),
			},
			expected: []any{"9.5", "0.25"},
		},
		{
			name: "nested map of big.Rat",
			input: map[string]any{
				"val1": new(big.Rat).SetFrac64(19, 2),
				"val2": new(big.Rat).SetFrac64(1, 2),
			},
			expected: map[string]any{
				"val1": "9.5",
				"val2": "0.5",
			},
		},
		{
			name: "complex nested structure",
			input: map[string]any{
				"list": []any{
					map[string]any{
						"rat": new(big.Rat).SetFrac64(3, 2),
					},
				},
			},
			expected: map[string]any{
				"list": []any{
					map[string]any{
						"rat": "1.5",
					},
				},
			},
		},
		{
			name: "slice of *big.Rat",
			input: []*big.Rat{
				new(big.Rat).SetFrac64(19, 2),
				new(big.Rat).SetFrac64(1, 4),
			},
			expected: []any{"9.5", "0.25"},
		},
		{
			name:     "slice of strings",
			input:    []string{"a", "b"},
			expected: []any{"a", "b"},
		},
		{
			name:     "byte slice (BYTES)",
			input:    []byte("hello"),
			expected: []byte("hello"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bigquery.NormalizeValue(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("NormalizeValue() = %v, want %v", got, tt.expected)
			}
		})
	}
}
