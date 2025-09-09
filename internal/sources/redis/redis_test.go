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

package redis_test

import (
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/redis"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlRedis(t *testing.T) {
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
					type: redis
					address:
					  - 127.0.0.1
			`,
			want: server.SourceConfigs{
				"my-redis-instance": redis.Config{
					Name:           "my-redis-instance",
					Type:           redis.SourceType,
					Address:        []string{"127.0.0.1"},
					ClusterEnabled: false,
					UseGCPIAM:      false,
				},
			},
		},
		{
			desc: "advanced example",
			in: `
			sources:
				my-redis-instance:
					type: redis
					address:
					  - 127.0.0.1
					password: my-pass
					database: 1
					useGCPIAM: true
					clusterEnabled: true
			`,
			want: server.SourceConfigs{
				"my-redis-instance": redis.Config{
					Name:           "my-redis-instance",
					Type:           redis.SourceType,
					Address:        []string{"127.0.0.1"},
					Password:       "my-pass",
					Database:       1,
					ClusterEnabled: true,
					UseGCPIAM:      true,
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
					type: redis
					project: my-project
					address:
					  - 127.0.0.1
					password: my-pass
					database: data
			`,
			err: "cannot unmarshal string into Go struct field .Sources of type int",
		},
		{
			desc: "extra field",
			in: `
			sources:
				my-redis-instance:
					type: redis
					project: my-project
					address:
					  - 127.0.0.1
					password: my-pass
					database: 1
			`,
			err: "unable to parse source \"my-redis-instance\" as \"redis\": [6:1] unknown field \"project\"",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-redis-instance:
					type: redis
			`,
			err: "unable to parse source \"my-redis-instance\" as \"redis\": Key: 'Config.Address' Error:Field validation for 'Address' failed on the 'required' tag",
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
