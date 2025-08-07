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

package elasticsearch_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/elasticsearch"
)

func TestParseFromYamlElasticsearch(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
sources:
  my-es-instance:
    kind: elasticsearch
    addresses:
      - http://localhost:9200
    apikey: somekey
`,

			want: server.SourceConfigs{
				"my-es-instance": elasticsearch.Config{
					Name:      "my-es-instance",
					Kind:      elasticsearch.SourceKind,
					Addresses: []string{"http://localhost:9200"},
					APIKey:    "somekey",
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			err := yaml.Unmarshal([]byte(tc.in), &got)
			if err != nil {
				t.Fatalf("failed to parse yaml: %v", err)
			}
			if diff := cmp.Diff(tc.want, got.Sources); diff != "" {
				t.Errorf("unexpected config diff (-want +got):\n%s", diff)
			}
		})
	}
}
