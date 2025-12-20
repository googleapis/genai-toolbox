// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutils

import (
	"fmt"
	"strings"
	"testing"

	dataproc "cloud.google.com/go/dataproc/v2/apiv1/dataprocpb"
	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/serverlessspark/createbatch"
	"google.golang.org/protobuf/testing/protocmp"
)

// RunParseFromYAMLTests runs a suite of tests for parsing tool configurations from YAML.
func RunParseFromYAMLTests(t *testing.T, kind string, newConfig func(c createbatch.Config) tools.ToolConfig) {
	t.Helper()
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	tcs := []struct {
		desc    string
		in      string
		want    server.ToolConfigs
		wantErr string
	}{
		{
			desc: "basic example",
			in: fmt.Sprintf(`
			tools:
				example_tool:
					kind: %s
					source: my-instance
					description: some description
			`, kind),
			want: server.ToolConfigs{
				"example_tool": newConfig(createbatch.Config{
					Name:         "example_tool",
					Kind:         kind,
					Source:       "my-instance",
					Description:  "some description",
					AuthRequired: []string{},
				}),
			},
		},
		{
			desc: "detailed config",
			in: fmt.Sprintf(`
			tools:
				example_tool:
					kind: %s
					source: my-instance
					description: some description
					runtimeConfig:
					  properties:
						  "spark.driver.memory": "1g"
					environmentConfig:
					  executionConfig:
						  networkUri: "my-network"
			`, kind),
			want: server.ToolConfigs{
				"example_tool": newConfig(createbatch.Config{
					Name:        "example_tool",
					Kind:        kind,
					Source:      "my-instance",
					Description: "some description",
					RuntimeConfig: &dataproc.RuntimeConfig{
						Properties: map[string]string{"spark.driver.memory": "1g"},
					},
					EnvironmentConfig: &dataproc.EnvironmentConfig{
						ExecutionConfig: &dataproc.ExecutionConfig{
							Network: &dataproc.ExecutionConfig_NetworkUri{NetworkUri: "my-network"},
						},
					},
					AuthRequired: []string{},
				}),
			},
		},
		{
			desc: "invalid runtime config",
			in: fmt.Sprintf(`
			tools:
				example_tool:
					kind: %s
					source: my-instance
					description: some description
					runtimeConfig:
					  invalidField: true
			`, kind),
			wantErr: "unmarshal runtimeConfig",
		},
		{
			desc: "invalid environment config",
			in: fmt.Sprintf(`
			tools:
				example_tool:
					kind: %s
					source: my-instance
					description: some description
					environmentConfig:
					  invalidField: true
			`, kind),
			wantErr: "unmarshal environmentConfig",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Tools server.ToolConfigs `yaml:"tools"`
			}{}
			err := yaml.UnmarshalContext(ctx, testutils.FormatYaml(tc.in), &got, yaml.Strict())
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error to contain %q, got %q", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}

			if diff := cmp.Diff(tc.want, got.Tools, protocmp.Transform()); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}
