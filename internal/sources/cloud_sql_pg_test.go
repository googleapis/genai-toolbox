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

package sources_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"gopkg.in/yaml.v3"
)

func TestParseFromYamlCloudSQLPg(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want sources.Configs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-pg-instance:
					kind: cloud-sql-postgres
					project: my-project
					region: my-region
					instance: my-instance
					database: my_db
			`,
			want: sources.Configs{
				"my-pg-instance": sources.CloudSQLPgConfig{
					Name:     "my-pg-instance",
					Kind:     sources.CloudSQLPgKind,
					Project:  "my-project",
					Region:   "my-region",
					Instance: "my-instance",
					Database: "my_db",
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources sources.Configs `yaml:"sources"`
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
