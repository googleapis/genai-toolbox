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

package memorystoreredis_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/memorystoreredis"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"gopkg.in/yaml.v3"
)

func TestParseFromYamlMemorystoreRedis(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "default setting",
			in: `
			sources:
				my-redis-instance:
					kind: memorystore-redis
					address: 127.0.0.1
					clusterEnabled: true
			`,
			want: server.SourceConfigs{
				"my-redis-instance": memorystoreredis.Config{
					Name:           "my-redis-instance",
					Kind:           memorystoreredis.SourceKind,
					Address:        "127.0.0.1",
					ClusterEnabled: true,
				},
			},
		},
		{
			desc: "advanced example",
			in: `
			sources:
				my-redis-instance:
					kind: memorystore-redis
					address: 127.0.0.1
					password: my-pass
					database: 1
					clusterEnabled: false
			`,
			want: server.SourceConfigs{
				"my-redis-instance": memorystoreredis.Config{
					Name:           "my-redis-instance",
					Kind:           memorystoreredis.SourceKind,
					Address:        "127.0.0.1",
					Password:       "my-pass",
					Database:       1,
					ClusterEnabled: false,
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
			desc: "invalid database",
			in: `
			sources:
				my-redis-instance:
					kind: memorystore-redis
					project: my-project
					address: 127.0.0.1
					password: my-pass
					clusterEnabled: 
			`,
			err: "cannot unmarshal string into Go struct field .Sources of type int",
		},
		{
			desc: "extra field",
			in: `
			sources:
				my-redis-instance:
					kind: memorystore-redis
					project: my-project
					address: 127.0.0.1
					password: my-pass
					database: 1
			`,
			err: "unable to parse as \"memorystore-redis\": [5:1] unknown field \"project\"",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-redis-instance:
					kind: memorystore-redis
			`,
			err: "unable to parse as \"memorystore-redis\": Key: 'Config.Address' Error:Field validation for 'Address' failed on the 'required' tag",
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
			if !strings.Contains(errStr, tc.err) {
				t.Fatalf("unexpected error: got %q, want %q", errStr, tc.err)
			}
		})
	}
}
