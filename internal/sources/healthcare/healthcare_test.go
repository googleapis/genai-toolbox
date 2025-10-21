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

package healthcare_test

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/healthcare"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlHealthcare(t *testing.T) {
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
						kind: healthcare
						project: my-project
						region: us-central1
						dataset: my-dataset
			`,
			want: server.SourceConfigs{
				"my-instance": healthcare.Config{
					Name:           "my-instance",
					Kind:           healthcare.SourceKind,
					Project:        "my-project",
					Region:         "us-central1",
					Dataset:        "my-dataset",
					UseClientOAuth: false,
				},
			},
		},
		{
			desc: "use client auth example",
			in: `
			sources:
				my-instance:
					kind: healthcare
					project: my-project
					region: us
					dataset: my-dataset
					useClientOAuth: true
			`,
			want: server.SourceConfigs{
				"my-instance": healthcare.Config{
					Name:           "my-instance",
					Kind:           healthcare.SourceKind,
					Project:        "my-project",
					Region:         "us",
					Dataset:        "my-dataset",
					UseClientOAuth: true,
				},
			},
		},
		{
			desc: "with allowed stores example",
			in: `
			sources:
				my-instance:
					kind: healthcare
					project: my-project
					region: us
					dataset: my-dataset
					allowedFhirStores:
						- my-fhir-store
					allowedDicomStores:
						- my-dicom-store1
						- my-dicom-store2
			`,
			want: server.SourceConfigs{
				"my-instance": healthcare.Config{
					Name:               "my-instance",
					Kind:               healthcare.SourceKind,
					Project:            "my-project",
					Region:             "us",
					Dataset:            "my-dataset",
					AllowedFHIRStores:  []string{"my-fhir-store"},
					AllowedDICOMStores: []string{"my-dicom-store1", "my-dicom-store2"},
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
					kind: healthcare
					project: my-project
					region: us-central1
					dataset: my-dataset
					foo: bar
			`,
			err: "unable to parse source \"my-instance\" as \"healthcare\": [2:1] unknown field \"foo\"\n   1 | dataset: my-dataset\n>  2 | foo: bar\n       ^\n   3 | kind: healthcare\n   4 | project: my-project\n   5 | region: us-central1",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-instance:
					kind: healthcare
					project: my-project
					region: us-central1
			`,
			err: `unable to parse source "my-instance" as "healthcare": Key: 'Config.Dataset' Error:Field validation for 'Dataset' failed on the 'required' tag`,
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
