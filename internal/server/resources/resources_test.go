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

package resources_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/server/resources"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/alloydbpg"
	"github.com/googleapis/genai-toolbox/internal/telemetry"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/stretchr/testify/assert"
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
	newToolsets := map[string]tools.Toolset{
		"example-toolset": {
			ToolsetConfig: tools.ToolsetConfig{
				Name: "example-toolset",
			},
			Tools: []*tools.Tool{},
		},
	}
	newPrompts := map[string]prompts.Prompt{"example-prompt": nil}
	newPromptsets := map[string]prompts.Promptset{
		"example-promptset": {
			PromptsetConfig: prompts.PromptsetConfig{
				Name: "example-promptset",
			},
			Prompts: []*prompts.Prompt{},
		},
	}
	resMgr := resources.NewResourceManager(newSources, newAuth, newEmbeddingModels, newTools, newToolsets, newPrompts, newPromptsets)

	gotSource, _ := resMgr.GetSource("example-source")
	if diff := cmp.Diff(gotSource, newSources["example-source"]); diff != "" {
		t.Errorf("error updating server, sources (-want +got):\n%s", diff)
	}

	gotAuthService, _ := resMgr.GetAuthService("example-auth")
	if diff := cmp.Diff(gotAuthService, newAuth["example-auth"]); diff != "" {
		t.Errorf("error updating server, authServices (-want +got):\n%s", diff)
	}

	gotTool, _ := resMgr.GetTool("example-tool")
	if diff := cmp.Diff(gotTool, newTools["example-tool"]); diff != "" {
		t.Errorf("error updating server, tools (-want +got):\n%s", diff)
	}

	gotToolset, _ := resMgr.GetToolset("example-toolset")
	if diff := cmp.Diff(gotToolset, newToolsets["example-toolset"]); diff != "" {
		t.Errorf("error updating server, toolset (-want +got):\n%s", diff)
	}

	gotPrompt, _ := resMgr.GetPrompt("example-prompt")
	if diff := cmp.Diff(gotPrompt, newPrompts["example-prompt"]); diff != "" {
		t.Errorf("error updating server, prompts (-want +got):\n%s", diff)
	}

	gotPromptset, _ := resMgr.GetPromptset("example-promptset")
	if diff := cmp.Diff(gotPromptset, newPromptsets["example-promptset"]); diff != "" {
		t.Errorf("error updating server, promptset (-want +got):\n%s", diff)
	}

	updateSource := map[string]sources.Source{
		"example-source2": &alloydbpg.Source{
			Config: alloydbpg.Config{
				Name: "example-alloydb-source2",
				Type: "alloydb-postgres",
			},
		},
	}

	resMgr.SetResources(updateSource, newAuth, newEmbeddingModels, newTools, newToolsets, newPrompts, newPromptsets)
	gotSource, _ = resMgr.GetSource("example-source2")
	if diff := cmp.Diff(gotSource, updateSource["example-source2"]); diff != "" {
		t.Errorf("error updating server, sources (-want +got):\n%s", diff)
	}
}

func TestCreateAndUpdatePrimitives(t *testing.T) {
	instrumentation, err := telemetry.CreateTelemetryInstrumentation("test-version")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	ctx := context.Background()
	ctx = util.WithInstrumentation(ctx, instrumentation)
	resMgr := resources.NewResourceManager(
		map[string]sources.Source{},
		map[string]auth.AuthService{},
		map[string]embeddingmodels.EmbeddingModel{},
		map[string]tools.Tool{},
		map[string]tools.Toolset{},
		map[string]prompts.Prompt{},
		map[string]prompts.Promptset{},
	)

	// create primitives
	sourceConfig := &testutils.MockSourceConfig{}
	err = resMgr.UpdateSource(ctx, "test-source", sourceConfig)
	assert.NoError(t, err)
	_, ok := resMgr.GetSource("test-source")
	if !ok {
		t.Fatalf("missing source")
	}

	asConfig := &testutils.MockAuthServiceConfig{}
	err = resMgr.UpdateAuthService(ctx, "test-auth-service", asConfig)
	assert.NoError(t, err)
	_, ok = resMgr.GetAuthService("test-auth-service")
	if !ok {
		t.Fatalf("missing auth service")
	}

	emConfig := &testutils.MockEmbeddingModelConfig{}
	err = resMgr.UpdateEmbeddingModel(ctx, "test-embedding-model", emConfig)
	assert.NoError(t, err)
	_, ok = resMgr.GetEmbeddingModel("test-embedding-model")
	if !ok {
		t.Fatalf("missing embedding model")
	}

	toolConfig := &testutils.MockToolConfig{}
	err = resMgr.UpdateTool(ctx, "test-tool", toolConfig)
	assert.NoError(t, err)
	_, ok = resMgr.GetTool("test-tool")
	if !ok {
		t.Fatalf("missing tool")
	}

	tsConfig := tools.ToolsetConfig{
		Name:      "test-toolset",
		ToolNames: []string{},
	}
	err = resMgr.UpdateToolset(ctx, "test-toolset", tsConfig, "test-version")
	assert.NoError(t, err)
	_, ok = resMgr.GetToolset("test-toolset")
	if !ok {
		t.Fatalf("missing toolset")
	}

	pConfig := &testutils.MockPromptConfig{}
	err = resMgr.UpdatePrompt(ctx, "test-prompt", pConfig)
	assert.NoError(t, err)
	_, ok = resMgr.GetPrompt("test-prompt")
	if !ok {
		t.Fatalf("missing prompt")
	}

	// Update primitives
	sourceConfig = &testutils.MockSourceConfig{Foo: "foo"}
	err = resMgr.UpdateSource(ctx, "test-source", sourceConfig)
	assert.NoError(t, err)
	source, ok := resMgr.GetSource("test-source")
	if !ok {
		t.Fatalf("missing source")
	}
	sc := source.ToConfig()
	if !reflect.DeepEqual(sc, sourceConfig) {
		t.Fatalf("update failed: got %+v, want %+v", sc, sourceConfig)
	}

	asConfig = &testutils.MockAuthServiceConfig{}
	err = resMgr.UpdateAuthService(ctx, "test-auth-service", asConfig)
	assert.NoError(t, err)
	as, ok := resMgr.GetAuthService("test-auth-service")
	if !ok {
		t.Fatalf("missing auth service")
	}
	asc := as.ToConfig()
	if !reflect.DeepEqual(asc, asConfig) {
		t.Fatalf("update failed: got %+v, want %+v", asc, asConfig)
	}

	emConfig = &testutils.MockEmbeddingModelConfig{}
	err = resMgr.UpdateEmbeddingModel(ctx, "test-embedding-model", emConfig)
	assert.NoError(t, err)
	em, ok := resMgr.GetEmbeddingModel("test-embedding-model")
	if !ok {
		t.Fatalf("missing embedding model")
	}
	emc := em.ToConfig()
	if !reflect.DeepEqual(emc, emConfig) {
		t.Fatalf("update failed: got %+v, want %+v", emc, emConfig)
	}

	toolConfig = &testutils.MockToolConfig{}
	err = resMgr.UpdateTool(ctx, "test-tool", toolConfig)
	assert.NoError(t, err)
	tool, ok := resMgr.GetTool("test-tool")
	if !ok {
		t.Fatalf("missing tool")
	}
	tc := tool.ToConfig()
	if !reflect.DeepEqual(tc, toolConfig) {
		t.Fatalf("update failed: got %+v, want %+v", tc, toolConfig)
	}

	tsConfig = tools.ToolsetConfig{
		Name:      "test-toolset",
		ToolNames: []string{"test-tool"},
	}
	err = resMgr.UpdateToolset(ctx, "test-toolset", tsConfig, "test-version")
	assert.NoError(t, err)
	ts, ok := resMgr.GetToolset("test-toolset")
	if !ok {
		t.Fatalf("missing toolset")
	}
	tsc := ts.ToConfig()
	if !reflect.DeepEqual(tsc, tsConfig) {
		t.Fatalf("update failed: got %+v, want %+v", tsc, tsConfig)
	}

	pConfig = &testutils.MockPromptConfig{}
	err = resMgr.UpdatePrompt(ctx, "test-prompt", pConfig)
	assert.NoError(t, err)
	p, ok := resMgr.GetPrompt("test-prompt")
	if !ok {
		t.Fatalf("missing prompt")
	}
	pc := p.ToConfig()
	if !reflect.DeepEqual(pc, pConfig) {
		t.Fatalf("update failed: got %+v, want %+v", pc, pConfig)
	}
}

func TestDeletePrimitives(t *testing.T) {
	resMgr := resources.NewResourceManager(
		map[string]sources.Source{"foo": testutils.MockSource{}},
		map[string]auth.AuthService{"foo": testutils.MockAuthService{}},
		map[string]embeddingmodels.EmbeddingModel{"foo": testutils.MockEmbeddingModel{}},
		map[string]tools.Tool{"foo": testutils.MockTool{}},
		map[string]tools.Toolset{"foo": tools.Toolset{}},
		map[string]prompts.Prompt{"foo": testutils.MockPrompt{}},
		map[string]prompts.Promptset{},
	)

	var deleted, found bool
	deleted = resMgr.DeleteSource("foo")
	if !deleted {
		t.Fatalf("expected delete to be successful")
	}
	_, found = resMgr.GetSource("foo")
	if found {
		t.Fatalf("found source but expected to be deleted")
	}
	deleted = resMgr.DeleteSource("nonexistent")
	if deleted {
		t.Fatalf("expected delete to be successful")
	}

	deleted = resMgr.DeleteAuthService("foo")
	if !deleted {
		t.Fatalf("expected delete to be successful")
	}
	_, found = resMgr.GetAuthService("foo")
	if found {
		t.Fatalf("found auth service but expected to be deleted")
	}
	deleted = resMgr.DeleteAuthService("nonexistent")
	if deleted {
		t.Fatalf("expected delete to be successful")
	}

	deleted = resMgr.DeleteEmbeddingModel("foo")
	if !deleted {
		t.Fatalf("expected delete to be successful")
	}
	_, found = resMgr.GetEmbeddingModel("foo")
	if found {
		t.Fatalf("found embedding model but expected to be deleted")
	}
	deleted = resMgr.DeleteEmbeddingModel("nonexistent")
	if deleted {
		t.Fatalf("expected delete to be successful")
	}

	deleted = resMgr.DeleteTool("foo")
	if !deleted {
		t.Fatalf("expected delete to be successful")
	}
	_, found = resMgr.GetTool("foo")
	if found {
		t.Fatalf("found tool but expected to be deleted")
	}
	deleted = resMgr.DeleteTool("nonexistent")
	if deleted {
		t.Fatalf("expected delete to be successful")
	}

	deleted = resMgr.DeleteToolset("foo")
	if !deleted {
		t.Fatalf("expected delete to be successful")
	}
	_, found = resMgr.GetToolset("foo")
	if found {
		t.Fatalf("found toolset but expected to be deleted")
	}
	deleted = resMgr.DeleteToolset("nonexistent")
	if deleted {
		t.Fatalf("expected delete to be successful")
	}

	deleted = resMgr.DeletePrompt("foo")
	if !deleted {
		t.Fatalf("expected delete to be successful")
	}
	_, found = resMgr.GetPrompt("foo")
	if found {
		t.Fatalf("found prompt but expected to be deleted")
	}
	deleted = resMgr.DeletePrompt("nonexistent")
	if deleted {
		t.Fatalf("expected delete to be successful")
	}
}

func TestGetPrimitives(t *testing.T) {
	resMgr := resources.NewResourceManager(
		map[string]sources.Source{"foo": testutils.MockSource{}},
		map[string]auth.AuthService{"foo": testutils.MockAuthService{}},
		map[string]embeddingmodels.EmbeddingModel{"foo": testutils.MockEmbeddingModel{}},
		map[string]tools.Tool{"foo": testutils.MockTool{}},
		map[string]tools.Toolset{"foo": tools.Toolset{}},
		map[string]prompts.Prompt{"foo": testutils.MockPrompt{}},
		map[string]prompts.Promptset{},
	)
	want := []string{"foo"}

	got := resMgr.GetSources()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected sources list: got %v, want %v", got, want)
	}

	got = resMgr.GetAuthServices()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected auth services list: got %v, want %v", got, want)
	}

	got = resMgr.GetEmbeddingModels()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected embedding models list: got %v, want %v", got, want)
	}

	got = resMgr.GetTools()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected tools list: got %v, want %v", got, want)
	}

	got = resMgr.GetToolsets()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected toolsets list: got %v, want %v", got, want)
	}

	got = resMgr.GetPrompts()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected prompts list: got %v, want %v", got, want)
	}
}
