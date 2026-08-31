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
	primMgr := primitives.NewPrimitiveManager(newSources, newAuth, newEmbeddingModels, newTools, newPrompts, newResources, newResourceTemplates, newGroups)

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

	primMgr.SetPrimitives(updateSource, newAuth, newEmbeddingModels, newTools, newPrompts, newResources, newResourceTemplates, newGroups)
	gotSource, _ = primMgr.GetSource("example-source2")
	if diff := cmp.Diff(gotSource, updateSource["example-source2"]); diff != "" {
		t.Errorf("error updating server, sources (-want +got):\n%s", diff)
	}
}

func TestGetResourceOrTemplateByURI(t *testing.T) {
	resourcesMap := map[string]resources.Resource{
		"res1": testutils.NewMockResource("res1", "file:///res1", "", "", "", nil, nil),
		"res2": testutils.NewMockResource("res2", "file:///res2", "", "", "", nil, nil),
	}
	templatesMap := map[string]resources.ResourceTemplate{
		"tmpl1": testutils.NewMockResourceTemplate("tmpl1", "file:///tmpl/{path}", "", "", "", nil),
		"tmpl2": testutils.NewMockResourceTemplate("tmpl2", "file:///other/{path}", "", "", "", nil),
	}

	// Create a group that only contains res1 and tmpl1
	g, err := group.GroupConfig{
		Name:                  "test_group",
		ResourceNames:         []string{"res1"},
		ResourceTemplateNames: []string{"tmpl1"},
	}.Initialize(nil, nil, resourcesMap, templatesMap)
	if err != nil {
		t.Fatalf("failed to init group: %v", err)
	}

	primMgr := primitives.NewPrimitiveManager(nil, nil, nil, nil, nil, resourcesMap, templatesMap, map[string]group.Group{"test_group": g})

	t.Run("Exact Match Resource", func(t *testing.T) {
		res, tmpl, params, err := primMgr.GetResourceOrTemplateByURI("file:///res1", g)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if res == nil || res.GetName() != "res1" {
			t.Errorf("expected res1, got %v", res)
		}
		if tmpl != nil {
			t.Errorf("expected nil template, got %v", tmpl)
		}
		if params != nil {
			t.Errorf("expected nil params, got %v", params)
		}
	})

	t.Run("Excluded Resource (Not in Group)", func(t *testing.T) {
		_, _, _, err := primMgr.GetResourceOrTemplateByURI("file:///res2", g)
		if err == nil {
			t.Fatal("expected error for resource not in group")
		}
	})

	t.Run("Template Match", func(t *testing.T) {
		res, tmpl, params, err := primMgr.GetResourceOrTemplateByURI("file:///tmpl/foo/bar.txt", g)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if res != nil {
			t.Errorf("expected nil resource, got %v", res)
		}
		if tmpl == nil || tmpl.GetName() != "tmpl1" {
			t.Errorf("expected tmpl1, got %v", tmpl)
		}
		if params["path"] != "foo/bar.txt" {
			t.Errorf("expected path param 'foo/bar.txt', got %v", params["path"])
		}
	})

	t.Run("Excluded Template (Not in Group)", func(t *testing.T) {
		_, _, _, err := primMgr.GetResourceOrTemplateByURI("file:///other/baz.txt", g)
		if err == nil {
			t.Fatal("expected error for template not in group")
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		_, _, _, err := primMgr.GetResourceOrTemplateByURI("file:///unknown", g)
		if err == nil {
			t.Fatal("expected error for unknown URI")
		}
	})
}
