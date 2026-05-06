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

package cloudstoragedownloadobject_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/tools/cloudstorage/cloudstoragedownloadobject"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

func TestParseFromYamlCloudStorageDownloadObject(t *testing.T) {
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
			name: download_tool
			type: cloud-storage-download-object
			source: my-gcs
			description: Download a Cloud Storage object
			`,
			want: server.ToolConfigs{
				"download_tool": cloudstoragedownloadobject.Config{
					Name:         "download_tool",
					Type:         "cloud-storage-download-object",
					Source:       "my-gcs",
					Description:  "Download a Cloud Storage object",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth requirements",
			in: `
			kind: tool
			name: secure_download
			type: cloud-storage-download-object
			source: prod-gcs
			description: Download with authentication
			authRequired:
				- google-auth-service
			`,
			want: server.ToolConfigs{
				"secure_download": cloudstoragedownloadobject.Config{
					Name:         "secure_download",
					Type:         "cloud-storage-download-object",
					Source:       "prod-gcs",
					Description:  "Download with authentication",
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
	called       bool
	gotDest      string
	gotOverwrite bool
}

func (m *mockSource) DownloadObject(ctx context.Context, bucket, object, destination string, overwrite bool) (map[string]any, error) {
	m.called = true
	m.gotDest = destination
	m.gotOverwrite = overwrite
	return map[string]any{"destination": destination, "bytes": int64(11), "contentType": "text/plain"}, nil
}

type mockSourceProvider struct {
	tools.SourceProvider
	source *mockSource
}

func (m *mockSourceProvider) GetSource(name string) (sources.Source, bool) {
	return m.source, true
}

func TestInvokeValidation(t *testing.T) {
	cfg := cloudstoragedownloadobject.Config{
		Name:        "download_tool",
		Type:        "cloud-storage-download-object",
		Source:      "my-gcs",
		Description: "Download",
	}
	tool, err := cfg.Initialize(nil)
	if err != nil {
		t.Fatalf("failed to initialize tool: %v", err)
	}
	validDest := filepath.Join(t.TempDir(), "out.bin")

	tcs := []struct {
		desc       string
		bucket     any
		object     any
		dest       any
		overwrite  any
		wantErr    bool
		wantSubstr string
		wantCalled bool
	}{
		{desc: "missing bucket", bucket: "", object: "o", dest: "/tmp/x", wantErr: true, wantSubstr: "bucket"},
		{desc: "missing object", bucket: "b", object: "", dest: "/tmp/x", wantErr: true, wantSubstr: "object"},
		{desc: "missing destination", bucket: "b", object: "o", dest: "", wantErr: true, wantSubstr: "destination"},
		{desc: "relative destination", bucket: "b", object: "o", dest: "relative/path", wantErr: true, wantSubstr: "destination"},
		{desc: "destination with traversal", bucket: "b", object: "o", dest: "/tmp/../etc/passwd", wantErr: true, wantSubstr: "destination"},
		{desc: "happy path", bucket: "b", object: "o", dest: validDest, overwrite: true, wantCalled: true},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			src := &mockSource{}
			resourceMgr := &mockSourceProvider{source: src}
			ov := false
			if b, ok := tc.overwrite.(bool); ok {
				ov = b
			}
			params := parameters.ParamValues{
				{Name: "bucket", Value: tc.bucket},
				{Name: "object", Value: tc.object},
				{Name: "destination", Value: tc.dest},
				{Name: "overwrite", Value: ov},
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
			if tc.wantCalled && !src.gotOverwrite {
				t.Errorf("overwrite flag not forwarded: got %v", src.gotOverwrite)
			}
		})
	}
}
