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

package cloudstoragecreatebucket_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/tools/cloudstorage/cloudstoragecreatebucket"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

func TestParseFromYamlCloudStorageCreateBucket(t *testing.T) {
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
			name: create_bucket_tool
			type: cloud-storage-create-bucket
			source: my-gcs
			description: Create a Cloud Storage bucket
			`,
			want: server.ToolConfigs{
				"create_bucket_tool": cloudstoragecreatebucket.Config{
					Name:         "create_bucket_tool",
					Type:         "cloud-storage-create-bucket",
					Source:       "my-gcs",
					Description:  "Create a Cloud Storage bucket",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth requirements",
			in: `
			kind: tool
			name: secure_create_bucket
			type: cloud-storage-create-bucket
			source: prod-gcs
			description: Create bucket with authentication
			authRequired:
				- google-auth-service
			`,
			want: server.ToolConfigs{
				"secure_create_bucket": cloudstoragecreatebucket.Config{
					Name:         "secure_create_bucket",
					Type:         "cloud-storage-create-bucket",
					Source:       "prod-gcs",
					Description:  "Create bucket with authentication",
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
	called                      bool
	gotBucket                   string
	gotLocation                 string
	gotUniformBucketLevelAccess bool
}

func (m *mockSource) CreateBucket(ctx context.Context, bucket, location string, uniformBucketLevelAccess bool) (map[string]any, error) {
	m.called = true
	m.gotBucket = bucket
	m.gotLocation = location
	m.gotUniformBucketLevelAccess = uniformBucketLevelAccess
	return map[string]any{"bucket": bucket, "created": true}, nil
}

type mockSourceProvider struct {
	tools.SourceProvider
	source *mockSource
}

func (m *mockSourceProvider) GetSource(name string) (sources.Source, bool) {
	return m.source, true
}

func initTool(t *testing.T) tools.Tool {
	t.Helper()
	cfg := cloudstoragecreatebucket.Config{
		Name:        "create_bucket_tool",
		Type:        "cloud-storage-create-bucket",
		Source:      "my-gcs",
		Description: "Create bucket",
	}
	tool, err := cfg.Initialize(nil)
	if err != nil {
		t.Fatalf("failed to initialize tool: %v", err)
	}
	return tool
}

func TestInvokeValidationAndForwarding(t *testing.T) {
	tcs := []struct {
		desc        string
		bucket      any
		location    any
		uniform     any
		wantErr     bool
		wantCalled  bool
		wantBucket  string
		wantLoc     string
		wantUniform bool
		wantSubstr  string
	}{
		{desc: "missing bucket", bucket: "", location: "US", uniform: false, wantErr: true, wantSubstr: "bucket"},
		{desc: "happy path", bucket: "b", location: "EU", uniform: true, wantCalled: true, wantBucket: "b", wantLoc: "EU", wantUniform: true},
		{desc: "omitted location", bucket: "b", uniform: false, wantCalled: true, wantBucket: "b"},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			tool := initTool(t)
			src := &mockSource{}
			resourceMgr := &mockSourceProvider{source: src}
			params := parameters.ParamValues{
				{Name: "bucket", Value: tc.bucket},
				{Name: "uniform_bucket_level_access", Value: tc.uniform},
			}
			if tc.location != nil {
				params = append(params, parameters.ParamValue{Name: "location", Value: tc.location})
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
			if src.gotBucket != tc.wantBucket || src.gotLocation != tc.wantLoc || src.gotUniformBucketLevelAccess != tc.wantUniform {
				t.Errorf("forwarded params = (%q, %q, %v)", src.gotBucket, src.gotLocation, src.gotUniformBucketLevelAccess)
			}
		})
	}
}
