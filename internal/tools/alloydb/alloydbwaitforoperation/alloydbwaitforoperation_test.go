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

package alloydbwaitforoperation_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	alloydbwaitforoperation "github.com/googleapis/genai-toolbox/internal/tools/alloydb/alloydbwaitforoperation"
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
				wait-for-thing:
					kind: alloydb-wait-for-operation
					source: some-source
					description: some description
					delay: 1s
					maxDelay: 5s
					multiplier: 1.5
					maxRetries: 5
			`,
			want: server.ToolConfigs{
				"wait-for-thing": alloydbwaitforoperation.Config{
					Name:         "wait-for-thing",
					Kind:         "alloydb-wait-for-operation",
					Source:       "some-source",
					Description:  "some description",
					AuthRequired: []string{},
					Delay:        "1s",
					MaxDelay:     "5s",
					Multiplier:   1.5,
					MaxRetries:   5,
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
