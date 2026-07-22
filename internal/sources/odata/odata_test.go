// Copyright 2026 Google LLC
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

package odata

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
)

func TestParseFromYamlOData(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
				kind: source
				name: my-instance
				type: odata
				baseUrl: https://example.com/OData/opu/odata
				timeout: 10s
				auth:
				  type: basic
				  username: testuser
				  password: testpassword
			`,
			want: map[string]sources.SourceConfig{
				"my-instance": Config{
					Name:    "my-instance",
					Type:    SourceType,
					BaseURL: "https://example.com/OData/opu/odata",
					Timeout: "10s",
					Auth: AuthConfig{
						Type:     "basic",
						Username: "testuser",
						Password: "testpassword",
					},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got, _, _, _, _, _, err := server.UnmarshalPrimitiveConfig(context.Background(), testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if !cmp.Equal(tc.want, got) {
				t.Fatalf("incorrect parse: want %v, got %v", tc.want, got)
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
				kind: source
				name: my-instance
				type: odata
				baseUrl: https://example.com/OData/opu/odata
				foo: bar
			`,
			err: "unknown field \"foo\"",
		},
		{
			desc: "missing required field baseUrl",
			in: `
				kind: source
				name: my-instance
				type: odata
			`,
			err: "failed on the 'required' tag",
		},
		{
			desc: "invalid auth type enum",
			in: `
				kind: source
				name: my-instance
				type: odata
				baseUrl: https://example.com/OData/opu/odata
				auth:
				  type: invalid_auth
			`,
			err: "failed on the 'oneof' tag",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, _, _, _, err := server.UnmarshalPrimitiveConfig(context.Background(), testutils.FormatYaml(tc.in))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.err)
			}
			if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("expected error containing %q, got %q", tc.err, err.Error())
			}
		})
	}
}
