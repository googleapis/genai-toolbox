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
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/bigquery"
	"github.com/googleapis/genai-toolbox/internal/testutils"
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
					location: us
			`,
			want: server.SourceConfigs{
				"my-instance": bigquery.Config{
					Name:     "my-instance",
					Kind:     bigquery.SourceKind,
					Project:  "my-project",
					Location: "us",
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

func TestValidateDatasetAccess(t *testing.T) {
	tests := []struct {
		name           string
		allowedDataset string
		requestDataset string
		wantError      bool
		errorContains  string
	}{
		// Empty configuration - no access allowed
		{
			name:           "empty config denies all access",
			allowedDataset: "",
			requestDataset: "BillingReport_test",
			wantError:      true,
			errorContains:  "no dataset access configured - all queries are denied",
		},
		{
			name:           "empty config denies any dataset",
			allowedDataset: "",
			requestDataset: "any_dataset",
			wantError:      true,
			errorContains:  "no dataset access configured - all queries are denied",
		},
		// Wildcard access - allows all datasets
		{
			name:           "wildcard allows BillingReport dataset",
			allowedDataset: "*",
			requestDataset: "BillingReport_test",
			wantError:      false,
		},
		{
			name:           "wildcard allows any dataset",
			allowedDataset: "*",
			requestDataset: "any_dataset_name",
			wantError:      false,
		},
		{
			name:           "wildcard allows dataset with special characters",
			allowedDataset: "*",
			requestDataset: "dataset_with_123_numbers",
			wantError:      false,
		},
		{
			name:           "wildcard allows empty dataset name",
			allowedDataset: "*",
			requestDataset: "",
			wantError:      false,
		},
		// Specific dataset access - exact match required
		{
			name:           "exact match allows access",
			allowedDataset: "BillingReport_wfhxhd0rrqwoo8tizt5yvw",
			requestDataset: "BillingReport_wfhxhd0rrqwoo8tizt5yvw",
			wantError:      false,
		},
		{
			name:           "exact match with different dataset denied",
			allowedDataset: "BillingReport_wfhxhd0rrqwoo8tizt5yvw",
			requestDataset: "BillingReport_pc_7h33wqtez_j_libvf4a",
			wantError:      true,
			errorContains:  "access denied to dataset 'BillingReport_pc_7h33wqtez_j_libvf4a', only 'BillingReport_wfhxhd0rrqwoo8tizt5yvw' is allowed",
		},
		{
			name:           "case sensitive dataset names",
			allowedDataset: "BillingReport_test",
			requestDataset: "billingreport_test",
			wantError:      true,
			errorContains:  "access denied to dataset 'billingreport_test', only 'BillingReport_test' is allowed",
		},
		{
			name:           "partial match denied",
			allowedDataset: "BillingReport_test",
			requestDataset: "BillingReport_test_extended",
			wantError:      true,
			errorContains:  "access denied to dataset 'BillingReport_test_extended', only 'BillingReport_test' is allowed",
		},
		{
			name:           "substring match denied",
			allowedDataset: "BillingReport_test_long",
			requestDataset: "BillingReport_test",
			wantError:      true,
			errorContains:  "access denied to dataset 'BillingReport_test', only 'BillingReport_test_long' is allowed",
		},
		// Edge cases
		{
			name:           "empty dataset request with specific allowed",
			allowedDataset: "BillingReport_test",
			requestDataset: "",
			wantError:      true,
			errorContains:  "access denied to dataset '', only 'BillingReport_test' is allowed",
		},
		{
			name:           "whitespace in dataset name",
			allowedDataset: "BillingReport_test",
			requestDataset: "BillingReport_test ",
			wantError:      true,
			errorContains:  "access denied to dataset 'BillingReport_test ', only 'BillingReport_test' is allowed",
		},
		{
			name:           "dataset with special characters allowed",
			allowedDataset: "dataset-with_special.chars123",
			requestDataset: "dataset-with_special.chars123",
			wantError:      false,
		},
		// Real-world dataset names
		{
			name:           "production dataset access",
			allowedDataset: "BillingReport_pc_7h33wqtez_j_libvf4a",
			requestDataset: "BillingReport_pc_7h33wqtez_j_libvf4a",
			wantError:      false,
		},
		{
			name:           "different production dataset denied",
			allowedDataset: "BillingReport_pc_7h33wqtez_j_libvf4a",
			requestDataset: "BillingReport_wfhxhd0rrqwoo8tizt5yvw",
			wantError:      true,
			errorContains:  "access denied to dataset 'BillingReport_wfhxhd0rrqwoo8tizt5yvw', only 'BillingReport_pc_7h33wqtez_j_libvf4a' is allowed",
		},
		// Unicode and international characters
		{
			name:           "unicode dataset name",
			allowedDataset: "dataset_测试_тест",
			requestDataset: "dataset_测试_тест",
			wantError:      false,
		},
		{
			name:           "unicode mismatch",
			allowedDataset: "dataset_测试",
			requestDataset: "dataset_тест",
			wantError:      true,
			errorContains:  "access denied to dataset 'dataset_тест', only 'dataset_测试' is allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a Source with the test configuration
			source := &bigquery.Source{
				Name:           "test-source",
				Kind:           bigquery.SourceKind,
				Client:         nil, // Not needed for validation tests
				Location:       "us-central1",
				AllowedDataset: tt.allowedDataset,
			}

			// Test the ValidateDatasetAccess method
			err := source.ValidateDatasetAccess(tt.requestDataset)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', but got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %s", err.Error())
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
