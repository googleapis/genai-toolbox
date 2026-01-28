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

package spanneradmin_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/spanneradmin"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlSpannerAdmin(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			kind: sources
			name: my-spanner-admin-instance
			type: spanner-admin
			`,
			want: map[string]sources.SourceConfig{
				"my-spanner-admin-instance": spanneradmin.Config{
					Name:           "my-spanner-admin-instance",
					Type:           spanneradmin.SourceType,
					UseClientOAuth: false,
				},
			},
		},
		{
			desc: "use client auth example",
			in: `
			kind: sources
			name: my-spanner-admin-instance
			type: spanner-admin
			useClientOAuth: true
			`,
			want: map[string]sources.SourceConfig{
				"my-spanner-admin-instance": spanneradmin.Config{
					Name:           "my-spanner-admin-instance",
					Type:           spanneradmin.SourceType,
					UseClientOAuth: true,
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got, _, _, _, _, _, err := server.UnmarshalResourceConfig(context.Background(), testutils.FormatYaml(tc.in))
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
	t.Parallel()
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "extra field",
			in: `
			kind: sources
			name: my-spanner-admin-instance
			type: spanner-admin
			project: test-project
			`,
			err: `error unmarshaling sources: unable to parse source "my-spanner-admin-instance" as "spanner-admin": [2:1] unknown field "project"
   1 | name: my-spanner-admin-instance
>  2 | project: test-project
       ^
   3 | type: spanner-admin`,
		},
		{
			desc: "missing required field",
			in: `
			kind: sources
			name: my-spanner-admin-instance
			useClientOAuth: true
			`,
			err: "error unmarshaling sources: missing 'type' field or it is not a string",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, _, _, _, err := server.UnmarshalResourceConfig(context.Background(), testutils.FormatYaml(tc.in))
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
