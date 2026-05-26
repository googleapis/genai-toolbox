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

package cloudstoragedeleteobject_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/tools/cloudstorage/cloudstoragedeleteobject"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

func TestParseFromYamlCloudStorageDeleteObject(t *testing.T) {
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
			name: delete_tool
			type: cloud-storage-delete-object
			source: my-gcs
			description: Delete a Cloud Storage object
			`,
			want: server.ToolConfigs{
				"delete_tool": cloudstoragedeleteobject.Config{
					Name:         "delete_tool",
					Type:         "cloud-storage-delete-object",
					Source:       "my-gcs",
					Description:  "Delete a Cloud Storage object",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth requirements",
			in: `
			kind: tool
			name: secure_delete
			type: cloud-storage-delete-object
			source: prod-gcs
			description: Delete with authentication
			authRequired:
				- google-auth-service
			`,
			want: server.ToolConfigs{
				"secure_delete": cloudstoragedeleteobject.Config{
					Name:         "secure_delete",
					Type:         "cloud-storage-delete-object",
					Source:       "prod-gcs",
					Description:  "Delete with authentication",
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

type mockSource struct {
	sources.Source
	called    bool
	gotBucket string
	gotObject string
}

func (m *mockSource) DeleteObject(ctx context.Context, bucket, object string) (map[string]any, error) {
	m.called = true
	m.gotBucket = bucket
	m.gotObject = object
	return map[string]any{"bucket": bucket, "object": object, "deleted": true}, nil
}

type mockSourceProvider struct {
	tools.SourceProvider
	source *mockSource
}

func (m *mockSourceProvider) GetSource(name string) (sources.Source, bool) {
	return m.source, true
}

func TestInvokeValidation(t *testing.T) {
	cfg := cloudstoragedeleteobject.Config{
		Name:        "delete_tool",
		Type:        "cloud-storage-delete-object",
		Source:      "my-gcs",
		Description: "Delete",
	}
	tool, err := cfg.Initialize(nil)
	if err != nil {
		t.Fatalf("failed to initialize tool: %v", err)
	}

	tcs := []struct {
		desc       string
		bucket     any
		object     any
		wantErr    bool
		wantSubstr string
		wantCalled bool
	}{
		{desc: "missing bucket", bucket: "", object: "o", wantErr: true, wantSubstr: "bucket"},
		{desc: "missing object", bucket: "b", object: "", wantErr: true, wantSubstr: "object"},
		{desc: "happy path", bucket: "b", object: "o", wantCalled: true},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			src := &mockSource{}
			resourceMgr := &mockSourceProvider{source: src}
			params := parameters.ParamValues{
				{Name: "bucket", Value: tc.bucket},
				{Name: "object", Value: tc.object},
			}
			_, toolErr := tool.Invoke(context.Background(), resourceMgr, params, "")
			if tc.wantErr {
				if toolErr == nil {
					t.Fatalf("expected error, got nil")
				}
				if _, ok := toolErr.(*util.AgentError); !ok {
					t.Fatalf("expected *AgentError, got %T: %v", toolErr, toolErr)
				}
				if !strings.Contains(toolErr.Error(), tc.wantSubstr) {
					t.Errorf("error %q does not contain %q", toolErr, tc.wantSubstr)
				}
				if src.called {
					t.Errorf("expected source not to be called on validation failure")
				}
				return
			}
			if toolErr != nil {
				t.Fatalf("unexpected error: %v", toolErr)
			}
			if src.called != tc.wantCalled {
				t.Errorf("called = %v, want %v", src.called, tc.wantCalled)
			}
			if src.gotBucket != tc.bucket || src.gotObject != tc.object {
				t.Errorf("forwarded params = (%q, %q)", src.gotBucket, src.gotObject)
			}
		})
	}
}
