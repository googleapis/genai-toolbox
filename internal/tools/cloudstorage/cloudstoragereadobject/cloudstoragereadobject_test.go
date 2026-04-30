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

package cloudstoragereadobject

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
)

func TestParseFromYamlCloudStorageReadObject(t *testing.T) {
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
			kind: tool
			name: read_object_tool
			type: cloud-storage-read-object
			source: my-gcs
			description: Read a Cloud Storage object
			`,
			want: server.ToolConfigs{
				"read_object_tool": Config{
					Name:         "read_object_tool",
					Type:         "cloud-storage-read-object",
					Source:       "my-gcs",
					Description:  "Read a Cloud Storage object",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth requirements",
			in: `
			kind: tool
			name: secure_read_object
			type: cloud-storage-read-object
			source: prod-gcs
			description: Read object with authentication
			authRequired:
				- google-auth-service
			`,
			want: server.ToolConfigs{
				"secure_read_object": Config{
					Name:         "secure_read_object",
					Type:         "cloud-storage-read-object",
					Source:       "prod-gcs",
					Description:  "Read object with authentication",
					AuthRequired: []string{"google-auth-service"},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, got, _, _, err := server.UnmarshalResourceConfig(ctx, testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}

func TestParseRange(t *testing.T) {
	tcs := []struct {
		in         string
		wantOffset int64
		wantLength int64
		wantErr    bool
	}{
		{in: "", wantOffset: 0, wantLength: -1},
		{in: "bytes=0-9", wantOffset: 0, wantLength: 10},
		{in: "bytes=10-19", wantOffset: 10, wantLength: 10},
		{in: "bytes=10-", wantOffset: 10, wantLength: -1},
		{in: "bytes=-5", wantOffset: -5, wantLength: -1},
		{in: "bytes=0-0", wantOffset: 0, wantLength: 1},

		{in: "garbage", wantErr: true},
		{in: "bytes=", wantErr: true},
		{in: "bytes=a-b", wantErr: true},
		{in: "bytes=-", wantErr: true},
		{in: "bytes=-0", wantErr: true},
		{in: "bytes=5-2", wantErr: true},
		{in: "bytes=-1-2", wantErr: true},
	}
	for _, tc := range tcs {
		t.Run(tc.in, func(t *testing.T) {
			offset, length, err := parseRange(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got offset=%d length=%d", offset, length)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if offset != tc.wantOffset || length != tc.wantLength {
				t.Fatalf("got (%d, %d), want (%d, %d)", offset, length, tc.wantOffset, tc.wantLength)
			}
		})
	}
}
