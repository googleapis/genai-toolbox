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

package lookerlistagents_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	lkr "github.com/googleapis/mcp-toolbox/internal/tools/looker/lookerlistagents"
	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

func TestParseFromYaml(t *testing.T) {
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
            name: test_tool
            type: looker-list-agents
            source: my-instance
            description: some description
                                `,
			want: server.ToolConfigs{
				"test_tool": lkr.Config{
					Name:         "test_tool",
					Type:         "looker-list-agents",
					Source:       "my-instance",
					Description:  "some description",
					AuthRequired: []string{},
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

func TestFailParseFromYaml(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "Invalid method",
			in: `
            kind: tool
            name: test_tool
            type: looker-list-agents
            source: my-instance
            method: GOT
            description: some description
                        `,
			err: "unknown field \"method\"",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, _, _, _, err := server.UnmarshalResourceConfig(ctx, testutils.FormatYaml(tc.in))
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			errStr := err.Error()
			if !strings.Contains(errStr, tc.err) {
				t.Fatalf("unexpected error string: got %q, want substring %q", errStr, tc.err)
			}
		})
	}
}

type MockSource struct {
	sources.Source
}

func (m MockSource) UseClientAuthorization() bool {
	return false
}

func (m MockSource) GetAuthTokenHeaderName() string {
	return "Authorization"
}

func (m MockSource) LookerApiSettings() *rtl.ApiSettings {
	return &rtl.ApiSettings{}
}

func (m MockSource) GetLookerSDK(string) (*v4.LookerSDK, error) {
	return &v4.LookerSDK{}, nil
}

type MockSourceProvider struct {
	tools.SourceProvider
	source MockSource
}

func (m MockSourceProvider) GetSource(name string) (sources.Source, bool) {
	return m.source, true
}

func TestInvokeValidation(t *testing.T) {
	resourceMgr := MockSourceProvider{source: MockSource{}}

	// No validation errors to mock for this simple tool that throws errors from Invoke directly
	_ = resourceMgr

}

func TestManifest(t *testing.T) {
	cfg := lkr.Config{
		Name:        "test_tool",
		Type:        "looker-list-agents",
		Source:      "my-instance",
		Description: "test description",
	}

	tool, err := cfg.Initialize(nil)
	if err != nil {
		t.Fatalf("failed to initialize tool: %v", err)
	}

	manifest := tool.Manifest()
	if manifest.Description != cfg.Description {
		t.Errorf("manifest description mismatch: got %q, want %q", manifest.Description, cfg.Description)
	}

	expectedParams := []string{}
	for _, p := range expectedParams {
		found := false
		for _, mp := range manifest.Parameters {
			if mp.Name == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected parameter %q not found in manifest", p)
		}
	}
}

func TestMcpManifest(t *testing.T) {
	cfg := lkr.Config{
		Name:        "test_tool",
		Type:        "looker-list-agents",
		Source:      "my-instance",
		Description: "test description",
	}

	tool, err := cfg.Initialize(nil)
	if err != nil {
		t.Fatalf("failed to initialize tool: %v", err)
	}

	mcp := tool.McpManifest()
	if mcp.Name != cfg.Name {
		t.Errorf("mcp manifest name mismatch: got %q, want %q", mcp.Name, cfg.Name)
	}

	properties := mcp.InputSchema.Properties
	expectedParams := []string{}
	for _, p := range expectedParams {
		if _, ok := properties[p]; !ok {
			t.Errorf("parameter %q not found in MCP properties", p)
		}
	}
}

func TestAnnotations(t *testing.T) {
	readOnlyFalse := false
	cfg := lkr.Config{
		Name:        "test_tool",
		Type:        "looker-list-agents",
		Source:      "my-instance",
		Description: "test description",
		Annotations: &tools.ToolAnnotations{
			ReadOnlyHint: &readOnlyFalse,
		},
	}

	tool, err := cfg.Initialize(nil)
	if err != nil {
		t.Fatalf("failed to initialize tool: %v", err)
	}

	mcp := tool.McpManifest()
	if mcp.Annotations == nil {
		t.Fatal("mcp manifest annotations is nil")
	}
	if mcp.Annotations.ReadOnlyHint == nil {
		t.Fatal("mcp manifest ReadOnlyHint is nil")
	}
	if *mcp.Annotations.ReadOnlyHint != true {
		t.Errorf("ReadOnlyHint should be true, got %v", *mcp.Annotations.ReadOnlyHint)
	}
}
