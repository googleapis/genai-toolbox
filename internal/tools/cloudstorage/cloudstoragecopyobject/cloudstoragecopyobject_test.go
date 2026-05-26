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

package cloudstoragecopyobject_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/tools/cloudstorage/cloudstoragecopyobject"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

func TestParseFromYamlCloudStorageCopyObject(t *testing.T) {
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
			name: copy_tool
			type: cloud-storage-copy-object
			source: my-gcs
			description: Copy a Cloud Storage object
			`,
			want: server.ToolConfigs{
				"copy_tool": cloudstoragecopyobject.Config{
					Name:         "copy_tool",
					Type:         "cloud-storage-copy-object",
					Source:       "my-gcs",
					Description:  "Copy a Cloud Storage object",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth requirements",
			in: `
			kind: tool
			name: secure_copy
			type: cloud-storage-copy-object
			source: prod-gcs
			description: Copy with authentication
			authRequired:
				- google-auth-service
			`,
			want: server.ToolConfigs{
				"secure_copy": cloudstoragecopyobject.Config{
					Name:         "secure_copy",
					Type:         "cloud-storage-copy-object",
					Source:       "prod-gcs",
					Description:  "Copy with authentication",
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
	called               bool
	gotSourceBucket      string
	gotSourceObject      string
	gotDestinationBucket string
	gotDestinationObject string
}

func (m *mockSource) CopyObject(ctx context.Context, sourceBucket, sourceObject, destinationBucket, destinationObject string) (map[string]any, error) {
	m.called = true
	m.gotSourceBucket = sourceBucket
	m.gotSourceObject = sourceObject
	m.gotDestinationBucket = destinationBucket
	m.gotDestinationObject = destinationObject
	return map[string]any{"sourceBucket": sourceBucket, "sourceObject": sourceObject, "destinationBucket": destinationBucket, "destinationObject": destinationObject}, nil
}

type mockSourceProvider struct {
	tools.SourceProvider
	source *mockSource
}

func (m *mockSourceProvider) GetSource(name string) (sources.Source, bool) {
	return m.source, true
}

func TestInvokeValidation(t *testing.T) {
	cfg := cloudstoragecopyobject.Config{
		Name:        "copy_tool",
		Type:        "cloud-storage-copy-object",
		Source:      "my-gcs",
		Description: "Copy",
	}
	tool, err := cfg.Initialize(nil)
	if err != nil {
		t.Fatalf("failed to initialize tool: %v", err)
	}

	tcs := []struct {
		desc              string
		sourceBucket      any
		sourceObject      any
		destinationBucket any
		destinationObject any
		wantErr           bool
		wantSubstr        string
		wantCalled        bool
	}{
		{desc: "missing source bucket", sourceBucket: "", sourceObject: "src", destinationBucket: "db", destinationObject: "dst", wantErr: true, wantSubstr: "source_bucket"},
		{desc: "missing source object", sourceBucket: "sb", sourceObject: "", destinationBucket: "db", destinationObject: "dst", wantErr: true, wantSubstr: "source_object"},
		{desc: "missing destination bucket", sourceBucket: "sb", sourceObject: "src", destinationBucket: "", destinationObject: "dst", wantErr: true, wantSubstr: "destination_bucket"},
		{desc: "missing destination object", sourceBucket: "sb", sourceObject: "src", destinationBucket: "db", destinationObject: "", wantErr: true, wantSubstr: "destination_object"},
		{desc: "happy path", sourceBucket: "sb", sourceObject: "src", destinationBucket: "db", destinationObject: "dst", wantCalled: true},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			src := &mockSource{}
			resourceMgr := &mockSourceProvider{source: src}
			params := parameters.ParamValues{
				{Name: "source_bucket", Value: tc.sourceBucket},
				{Name: "source_object", Value: tc.sourceObject},
				{Name: "destination_bucket", Value: tc.destinationBucket},
				{Name: "destination_object", Value: tc.destinationObject},
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
			if src.gotSourceBucket != tc.sourceBucket || src.gotSourceObject != tc.sourceObject || src.gotDestinationBucket != tc.destinationBucket || src.gotDestinationObject != tc.destinationObject {
				t.Errorf("forwarded params = (%q, %q, %q, %q)", src.gotSourceBucket, src.gotSourceObject, src.gotDestinationBucket, src.gotDestinationObject)
			}
		})
	}
}
