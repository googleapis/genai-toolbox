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

package cloudstoragegetobjectmetadata_test

import (
	"context"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/tools/cloudstorage/cloudstoragegetobjectmetadata"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

func TestParseFromYamlCloudStorageGetObjectMetadata(t *testing.T) {
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
			name: metadata_tool
			type: cloud-storage-get-object-metadata
			source: my-gcs
			description: Get object metadata
			`,
			want: server.ToolConfigs{
				"metadata_tool": cloudstoragegetobjectmetadata.Config{
					Name:         "metadata_tool",
					Type:         "cloud-storage-get-object-metadata",
					Source:       "my-gcs",
					Description:  "Get object metadata",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth requirements",
			in: `
			kind: tool
			name: secure_metadata
			type: cloud-storage-get-object-metadata
			source: prod-gcs
			description: Get metadata with authentication
			authRequired:
				- google-auth-service
			`,
			want: server.ToolConfigs{
				"secure_metadata": cloudstoragegetobjectmetadata.Config{
					Name:         "secure_metadata",
					Type:         "cloud-storage-get-object-metadata",
					Source:       "prod-gcs",
					Description:  "Get metadata with authentication",
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
	called bool
}

func (m *mockSource) GetObjectMetadata(ctx context.Context, bucket, object string) (*storage.ObjectAttrs, error) {
	m.called = true
	return &storage.ObjectAttrs{Bucket: bucket, Name: object, ContentType: "text/plain", Size: 11}, nil
}

type mockSourceProvider struct {
	tools.SourceProvider
	source *mockSource
}

func (m *mockSourceProvider) GetSource(name string) (sources.Source, bool) {
	return m.source, true
}

func TestInvokeValidation(t *testing.T) {
	cfg := cloudstoragegetobjectmetadata.Config{
		Name:        "metadata_tool",
		Type:        "cloud-storage-get-object-metadata",
		Source:      "my-gcs",
		Description: "Get object metadata",
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
		wantCalled bool
		wantSubstr string
	}{
		{desc: "missing bucket", bucket: "", object: "foo", wantErr: true, wantSubstr: "bucket"},
		{desc: "missing object", bucket: "b", object: "", wantErr: true, wantSubstr: "object"},
		{desc: "happy path", bucket: "b", object: "o", wantErr: false, wantCalled: true},
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
		})
	}
}
