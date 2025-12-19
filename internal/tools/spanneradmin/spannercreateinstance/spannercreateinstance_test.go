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

package spannercreateinstance_test

import (
	"context"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/spanneradmin/spannercreateinstance"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

func TestParseFromYaml(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		desc string
		in   string
		want server.ToolConfigs
	}{
		{
			desc: "basic example",
			in: `
			tools:
				create-instance-tool:
					kind: spanner-create-instance
					description: a test description
					source: a-source
			`,
			want: server.ToolConfigs{
				"create-instance-tool": spannercreateinstance.Config{
					Name:         "create-instance-tool",
					Kind:         "spanner-create-instance",
					Description:  "a test description",
					Source:       "a-source",
					AuthRequired: []string{},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Tools server.ToolConfigs `yaml:"tools"`
			}{}
			// Parse contents
			err := yaml.UnmarshalContext(ctx, testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got.Tools); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}

func TestInvoke_NodeCountAndProcessingUnitsValidation(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		params  parameters.ParamValues
		wantErr string
	}{
		{
			name: "Both positive",
			params: parameters.ParamValues{
				{Name: "nodeCount", Value: 1},
				{Name: "processingUnits", Value: 1000},
			},
			wantErr: "one of nodeCount or processingUnits must be positive, and the other must be 0",
		},
		{
			name: "Both zero",
			params: parameters.ParamValues{
				{Name: "nodeCount", Value: 0},
				{Name: "processingUnits", Value: 0},
			},
			wantErr: "one of nodeCount or processingUnits must be positive, and the other must be 0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tool := spannercreateinstance.Tool{}
			_, err := tool.Invoke(context.Background(), nil, tc.params, "")
			if err == nil || err.Error() != tc.wantErr {
				t.Errorf("Invoke() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
