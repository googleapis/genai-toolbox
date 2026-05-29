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

package tools

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

// stubConfig and stubTool exercise the path of embedding BaseTool with only
// the extra methods (Invoke, ToConfig) needed to satisfy the Tool interface.
// Real tools embed ConfigBase in their Config; the stub does too so it
// satisfies ToolMeta if/when wired into BaseTool.
type stubConfig struct {
	ConfigBase
}

func (stubConfig) ToolConfigType() string { return "stub" }
func (stubConfig) Initialize(map[string]sources.Source) (Tool, error) {
	return nil, nil
}

type stubTool struct {
	BaseTool
}

func (stubTool) Invoke(_ context.Context, _ SourceProvider, _ parameters.ParamValues, _ AccessToken) (any, util.ToolboxError) {
	return nil, nil
}

func (stubTool) ToConfig() ToolConfig { return stubConfig{} }

// Compile-time check: embedding BaseTool plus Invoke + ToConfig satisfies Tool.
var _ Tool = stubTool{}

// Compile-time check: ConfigBase satisfies ToolMeta on its own.
var _ ToolMeta = ConfigBase{}

func newBaseTool() (BaseTool, Manifest) {
	cfg := ConfigBase{
		Name:           "my-tool",
		Description:    "my tool description",
		AuthRequired:   []string{"google"},
		ScopesRequired: []string{"scope-a", "scope-b"},
	}
	manifest := Manifest{
		Description:  "manifest description",
		AuthRequired: []string{"google"},
	}
	b := BaseTool{
		cfg:              cfg,
		annotations:      NewReadOnlyAnnotations(),
		metadata:         manifest,
		StaticParameters: parameters.Parameters{parameters.NewStringParameter("p1", "first param")},
	}
	return b, manifest
}

func TestBaseToolGetters(t *testing.T) {
	b, wantManifest := newBaseTool()

	if got, want := b.GetName(), "my-tool"; got != want {
		t.Errorf("GetName() = %q, want %q", got, want)
	}
	if got, want := b.GetDescription(), "my tool description"; got != want {
		t.Errorf("GetDescription() = %q, want %q", got, want)
	}
	if diff := cmp.Diff([]string{"google"}, b.GetAuthRequired()); diff != "" {
		t.Errorf("GetAuthRequired() mismatch (-want +got):\n%s", diff)
	}
	got := b.GetAnnotations()
	if got == nil || got.ReadOnlyHint == nil || !*got.ReadOnlyHint {
		t.Errorf("GetAnnotations() = %+v, want ReadOnlyHint=true", got)
	}
	if diff := cmp.Diff(wantManifest, b.Manifest()); diff != "" {
		t.Errorf("Manifest() mismatch (-want +got):\n%s", diff)
	}
	if p := b.GetParameters(); len(p) != 1 || p[0].GetName() != "p1" {
		t.Errorf("GetParameters() = %+v, want one param named p1", p)
	}
	if diff := cmp.Diff([]string{"scope-a", "scope-b"}, b.GetScopesRequired()); diff != "" {
		t.Errorf("GetScopesRequired() mismatch (-want +got):\n%s", diff)
	}
}

func TestBaseToolAuthorized(t *testing.T) {
	tcs := []struct {
		desc         string
		authRequired []string
		verified     []string
		want         bool
	}{
		{"empty required is always authorized", nil, nil, true},
		{"empty required ignores verified", nil, []string{"foo"}, true},
		{"verified includes required", []string{"google"}, []string{"google", "github"}, true},
		{"verified missing required", []string{"google"}, []string{"github"}, false},
		{"verified empty when required non-empty", []string{"google"}, nil, false},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			b := BaseTool{cfg: ConfigBase{AuthRequired: tc.authRequired}}
			if got := b.Authorized(tc.verified); got != tc.want {
				t.Errorf("Authorized(%v) = %v, want %v", tc.verified, got, tc.want)
			}
		})
	}
}

func TestBaseToolRequiresClientAuthorization(t *testing.T) {
	b := BaseTool{}
	got, err := b.RequiresClientAuthorization(nil)
	if err != nil {
		t.Fatalf("RequiresClientAuthorization() error = %v", err)
	}
	if got {
		t.Errorf("RequiresClientAuthorization() = true, want false")
	}
}

func TestBaseToolGetAuthTokenHeaderName(t *testing.T) {
	b := BaseTool{}
	got, err := b.GetAuthTokenHeaderName(nil)
	if err != nil {
		t.Fatalf("GetAuthTokenHeaderName() error = %v", err)
	}
	if got != "Authorization" {
		t.Errorf("GetAuthTokenHeaderName() = %q, want %q", got, "Authorization")
	}
}

func TestBaseToolEmbedParamsPassthrough(t *testing.T) {
	b := BaseTool{
		StaticParameters: parameters.Parameters{parameters.NewStringParameter("p1", "first")},
	}
	values := parameters.ParamValues{{Name: "p1", Value: "hello"}}
	got, err := b.EmbedParams(context.Background(), values, map[string]embeddingmodels.EmbeddingModel{})
	if err != nil {
		t.Fatalf("EmbedParams() error = %v", err)
	}
	if diff := cmp.Diff(values, got); diff != "" {
		t.Errorf("EmbedParams() mismatch (-want +got):\n%s", diff)
	}
}
