// Copyright 2025 Google LLC
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

package primitives_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/auth"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/group"
	"github.com/googleapis/mcp-toolbox/internal/prompts"
	"github.com/googleapis/mcp-toolbox/internal/resources"
	"github.com/googleapis/mcp-toolbox/internal/server/primitives"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/sources/alloydbpg"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
)

func TestUpdateServer(t *testing.T) {
	newSources := map[string]sources.Source{
		"example-source": &alloydbpg.Source{
			Config: alloydbpg.Config{
				Name: "example-alloydb-source",
				Type: "alloydb-postgres",
			},
		},
	}
	newAuth := map[string]auth.AuthService{"example-auth": nil}
	newEmbeddingModels := map[string]embeddingmodels.EmbeddingModel{"example-model": nil}
	newTools := map[string]tools.Tool{"example-tool": nil}
	newPrompts := map[string]prompts.Prompt{"example-prompt": testutils.NewMockPrompt("example-prompt", "", prompts.Arguments{})}
	newGroups := map[string]group.Group{
		"example-toolset": group.NewGroup(group.GroupConfig{Name: "example-toolset", ToolNames: []string{"example-tool"}}),
	}
	newResources := map[string]resources.Resource{"example-resource": nil}
	newResourceTemplates := map[string]resources.ResourceTemplate{"example-template": nil}
	primMgr := primitives.NewPrimitiveManager(newSources, newAuth, newEmbeddingModels, newTools, newPrompts, newResources,newResourceTemplates, newGroups)

	gotSource, _ := primMgr.GetSource("example-source")
	if diff := cmp.Diff(gotSource, newSources["example-source"]); diff != "" {
		t.Errorf("error updating server, sources (-want +got):\n%s", diff)
	}

	gotAuthService, _ := primMgr.GetAuthService("example-auth")
	if diff := cmp.Diff(gotAuthService, newAuth["example-auth"]); diff != "" {
		t.Errorf("error updating server, authServices (-want +got):\n%s", diff)
	}

	gotResource, _ := primMgr.GetResource("example-resource")
	if diff := cmp.Diff(gotResource, newResources["example-resource"]); diff != "" {
		t.Errorf("error updating server, resources (-want +got):\n%s", diff)
	}

	gotTool, _ := primMgr.GetTool("example-tool")
	if diff := cmp.Diff(gotTool, newTools["example-tool"]); diff != "" {
		t.Errorf("error updating server, tools (-want +got):\n%s", diff)
	}

	wantGroup := newGroups["example-toolset"]
	gotGroup, ok := primMgr.GetGroup("example-toolset")
	if !ok {
		t.Fatal("expected group \"example-toolset\" to exist")
	}
	if diff := cmp.Diff(wantGroup, gotGroup, cmp.AllowUnexported(group.Group{})); diff != "" {
		t.Errorf("error updating server, group (-want +got):\n%s", diff)
	}

	gotPrompt, _ := primMgr.GetPrompt("example-prompt")
	if diff := cmp.Diff(gotPrompt, newPrompts["example-prompt"], cmp.AllowUnexported(testutils.MockPrompt{})); diff != "" {
		t.Errorf("error updating server, prompts (-want +got):\n%s", diff)
	}

	gotTemplate, _ := primMgr.GetResourceTemplate("example-template")
	if diff := cmp.Diff(gotTemplate, newResourceTemplates["example-template"]); diff != "" {
		t.Errorf("error updating server, resource templates (-want +got):\n%s", diff)
	}

	updateSource := map[string]sources.Source{
		"example-source2": &alloydbpg.Source{
			Config: alloydbpg.Config{
				Name: "example-alloydb-source2",
				Type: "alloydb-postgres",
			},
		},
	}

	primMgr.SetPrimitives(updateSource, newAuth, newEmbeddingModels, newTools, newPrompts, newResources,newResourceTemplates, newGroups)
	gotSource, _ = primMgr.GetSource("example-source2")
	if diff := cmp.Diff(gotSource, updateSource["example-source2"]); diff != "" {
		t.Errorf("error updating server, sources (-want +got):\n%s", diff)
	}
}

// TestPrimitiveManager_GetResourceOrTemplateByURI verifies that the primitive
// manager can correctly resolve exact URI matches for static resources, or
// fallback to matching and extracting parameters for URI templates.
func TestPrimitiveManager_GetResourceOrTemplateByURI(t *testing.T) {
	primMgr := primitives.NewPrimitiveManager(nil, nil, nil, nil, nil, nil, nil, nil)
	
	resourcesMap := map[string]resources.Resource{
		"static-res": testutils.NewMockResource("static-res", "file:///mock/resource/1", "", "", "", nil, nil),
	}
	templatesMap := map[string]resources.ResourceTemplate{
		"test-tmpl": testutils.NewMockResourceTemplate("test-tmpl", "file:///logs/{path}", "", "", "", nil),
	}

	
	gConfig := group.GroupConfig{
		Name: "",
		ResourceNames: []string{"static-res"},
		ResourceTemplateNames: []string{"test-tmpl"},
	}
	groups := map[string]group.Group{"": group.NewGroup(gConfig)}
	
	primMgr.SetPrimitives(nil, nil, nil, nil, nil, resourcesMap, templatesMap, groups)
	g, _ := primMgr.GetGroup("")

	// Test matching static resource
	res, tmpl, params, err := primMgr.GetResourceOrTemplateByURI("file:///mock/resource/1", g)
	if err != nil {
		t.Fatalf("GetResourceOrTemplateByURI failed for static resource: %v", err)
	}
	if res == nil {
		t.Errorf("Expected to find static resource, got nil")
	}
	if tmpl != nil {
		t.Errorf("Expected template to be nil for static resource, got %v", tmpl)
	}
	if params != nil {
		t.Errorf("Expected params to be nil for static resource, got %v", params)
	}

	// Test matching template
	res, tmpl, params, err = primMgr.GetResourceOrTemplateByURI("file:///logs/dynamic/dir/file.log", g)
	if err != nil {
		t.Fatalf("GetResourceOrTemplateByURI failed for template match: %v", err)
	}
	if res != nil {
		t.Errorf("Expected resource to be nil for template match, got %v", res)
	}
	if tmpl == nil {
		t.Errorf("Expected to find template, got nil")
	}
	if params == nil || params["path"] != "dynamic/dir/file.log" {
		t.Errorf("Expected params to contain path=dynamic/dir/file.log, got %v", params)
	}

	// Test no match
	_, _, _, err = primMgr.GetResourceOrTemplateByURI("http://example.com", g)
	if err == nil {
		t.Errorf("Expected error for unmatched URI, got nil")
	}

	// Test cross-group boundary isolation
	isolatedConfig := group.GroupConfig{
		Name: "isolated",
	}
	isolatedGroup := group.NewGroup(isolatedConfig)
	_, _, _, err = primMgr.GetResourceOrTemplateByURI("file:///mock/resource/1", isolatedGroup)
	if err == nil {
		t.Errorf("Expected error when requesting static resource outside of group boundary")
	}
	_, _, _, err = primMgr.GetResourceOrTemplateByURI("file:///logs/dynamic/dir/file.log", isolatedGroup)
	if err == nil {
		t.Errorf("Expected error when requesting template outside of group boundary")
	}
}
