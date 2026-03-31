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

package resources

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"sync"

	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ResourceManager contains available resources for the server. Should be initialized with NewResourceManager().
type ResourceManager struct {
	mu              sync.RWMutex
	sources         map[string]sources.Source
	authServices    map[string]auth.AuthService
	embeddingModels map[string]embeddingmodels.EmbeddingModel
	tools           map[string]tools.Tool
	toolsets        map[string]tools.Toolset
	prompts         map[string]prompts.Prompt
	promptsets      map[string]prompts.Promptset
}

func NewResourceManager(
	sourcesMap map[string]sources.Source,
	authServicesMap map[string]auth.AuthService,
	embeddingModelsMap map[string]embeddingmodels.EmbeddingModel,
	toolsMap map[string]tools.Tool, toolsetsMap map[string]tools.Toolset,
	promptsMap map[string]prompts.Prompt, promptsetsMap map[string]prompts.Promptset,

) *ResourceManager {
	resourceMgr := &ResourceManager{
		mu:              sync.RWMutex{},
		sources:         sourcesMap,
		authServices:    authServicesMap,
		embeddingModels: embeddingModelsMap,
		tools:           toolsMap,
		toolsets:        toolsetsMap,
		prompts:         promptsMap,
		promptsets:      promptsetsMap,
	}

	return resourceMgr
}

func (r *ResourceManager) SetResources(sourcesMap map[string]sources.Source, authServicesMap map[string]auth.AuthService, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel, toolsMap map[string]tools.Tool, toolsetsMap map[string]tools.Toolset, promptsMap map[string]prompts.Prompt, promptsetsMap map[string]prompts.Promptset) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources = sourcesMap
	r.authServices = authServicesMap
	r.embeddingModels = embeddingModelsMap
	r.tools = toolsMap
	r.toolsets = toolsetsMap
	r.prompts = promptsMap
	r.promptsets = promptsetsMap
}

func (r *ResourceManager) GetSources() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Sorted(maps.Keys(r.sources))
}

func (r *ResourceManager) GetAuthServices() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Sorted(maps.Keys(r.authServices))
}

func (r *ResourceManager) GetEmbeddingModels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Sorted(maps.Keys(r.embeddingModels))
}

func (r *ResourceManager) GetTools() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Sorted(maps.Keys(r.tools))
}

func (r *ResourceManager) GetToolsets() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Sorted(maps.Keys(r.toolsets))
}

func (r *ResourceManager) GetPrompts() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Sorted(maps.Keys(r.prompts))
}

func (r *ResourceManager) GetSource(sourceName string) (sources.Source, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	source, ok := r.sources[sourceName]
	return source, ok
}

func (r *ResourceManager) GetAuthService(authServiceName string) (auth.AuthService, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	authService, ok := r.authServices[authServiceName]
	return authService, ok
}

func (r *ResourceManager) GetEmbeddingModel(embeddingModelName string) (embeddingmodels.EmbeddingModel, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	model, ok := r.embeddingModels[embeddingModelName]
	return model, ok
}

func (r *ResourceManager) GetTool(toolName string) (tools.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[toolName]
	return tool, ok
}

func (r *ResourceManager) GetToolset(toolsetName string) (tools.Toolset, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	toolset, ok := r.toolsets[toolsetName]
	return toolset, ok
}

func (r *ResourceManager) GetPrompt(promptName string) (prompts.Prompt, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	prompt, ok := r.prompts[promptName]
	return prompt, ok
}

func (r *ResourceManager) GetPromptset(promptsetName string) (prompts.Promptset, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	promptset, ok := r.promptsets[promptsetName]
	return promptset, ok
}

func (r *ResourceManager) GetAuthServiceMap() map[string]auth.AuthService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copiedMap := make(map[string]auth.AuthService, len(r.authServices))
	for k, v := range r.authServices {
		copiedMap[k] = v
	}
	return copiedMap
}

func (r *ResourceManager) GetEmbeddingModelMap() map[string]embeddingmodels.EmbeddingModel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copiedMap := make(map[string]embeddingmodels.EmbeddingModel, len(r.embeddingModels))
	for k, v := range r.embeddingModels {
		copiedMap[k] = v
	}
	return copiedMap
}

func (r *ResourceManager) GetToolsMap() map[string]tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copiedMap := make(map[string]tools.Tool, len(r.tools))
	for k, v := range r.tools {
		copiedMap[k] = v
	}
	return copiedMap
}

func (r *ResourceManager) GetPromptsMap() map[string]prompts.Prompt {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copiedMap := make(map[string]prompts.Prompt, len(r.prompts))
	for k, v := range r.prompts {
		copiedMap[k] = v
	}
	return copiedMap
}

// UpdateSource creates or update source.
func (r *ResourceManager) UpdateSource(ctx context.Context, name string, config sources.SourceConfig) error {
	primitive, exist := r.GetSource(name)
	instrumentation, err := util.InstrumentationFromContext(ctx)
	if err != nil {
		return err
	}

	if exist {
		curConfig := primitive.ToConfig()
		// if config remains the same, return
		// if diff, re-initialize the primitive
		if reflect.DeepEqual(config, curConfig) {
			return nil
		}
	}

	childCtx, span := instrumentation.Tracer.Start(
		ctx,
		"toolbox/server/source/init",
		trace.WithAttributes(attribute.String("source_type", config.SourceConfigType())),
		trace.WithAttributes(attribute.String("source_name", name)),
	)
	defer span.End()
	s, err := config.Initialize(childCtx, instrumentation.Tracer)
	if err != nil {
		return fmt.Errorf("unable to initialize source %q: %w", name, err)
	}
	r.sources[name] = s
	return nil
}

// UpdateAuthService creates or update auth service.
func (r *ResourceManager) UpdateAuthService(ctx context.Context, name string, config auth.AuthServiceConfig) error {
	primitive, exist := r.GetAuthService(name)
	instrumentation, err := util.InstrumentationFromContext(ctx)
	if err != nil {
		return err
	}

	if exist {
		curConfig := primitive.ToConfig()
		// if config remains the same, return
		// if diff, re-initialize the primitive
		if reflect.DeepEqual(config, curConfig) {
			return nil
		}
	}

	_, span := instrumentation.Tracer.Start(
		ctx,
		"toolbox/server/auth/init",
		trace.WithAttributes(attribute.String("auth_type", config.AuthServiceConfigType())),
		trace.WithAttributes(attribute.String("auth_name", name)),
	)
	defer span.End()
	as, err := config.Initialize()
	if err != nil {
		return fmt.Errorf("unable to initialize auth service %q: %w", name, err)
	}
	r.authServices[name] = as
	return nil
}

// UpdateEmbeddingModel creates or update embedding model.
func (r *ResourceManager) UpdateEmbeddingModel(ctx context.Context, name string, config embeddingmodels.EmbeddingModelConfig) error {
	primitive, exist := r.GetEmbeddingModel(name)
	instrumentation, err := util.InstrumentationFromContext(ctx)
	if err != nil {
		return err
	}

	if exist {
		curConfig := primitive.ToConfig()
		// if config remains the same, return
		// if diff, re-initialize the primitive
		if reflect.DeepEqual(config, curConfig) {
			return nil
		}
	}
	_, span := instrumentation.Tracer.Start(
		ctx,
		"toolbox/server/embeddingmodel/init",
		trace.WithAttributes(attribute.String("model_type", config.EmbeddingModelConfigType())),
		trace.WithAttributes(attribute.String("model_name", name)),
	)
	defer span.End()
	em, err := config.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("unable to initialize embedding model %q: %w", name, err)
	}
	r.embeddingModels[name] = em
	return nil
}

// removeFirstFromToolset removes a target tool from toolset
func removeFirstFromToolset(ts tools.Toolset, target string) tools.Toolset {
	for i, tool := range ts.Tools {
		if tool != nil && (*tool) != nil && (*tool).McpManifest().Name == target {

			copy(ts.Tools[i:], ts.Tools[i+1:])

			// Set the very last element to nil to prevent a memory leak
			ts.Tools[len(ts.Tools)-1] = nil
			ts.Tools = ts.Tools[:len(ts.Tools)-1]
			return ts
		}
	}

	return ts
}

// UpdateTool creates or update tool.
func (r *ResourceManager) UpdateTool(ctx context.Context, name string, config tools.ToolConfig) error {
	primitive, exist := r.GetTool(name)
	instrumentation, err := util.InstrumentationFromContext(ctx)
	if err != nil {
		return err
	}

	defaultToolset := r.toolsets[""]
	if exist {
		curConfig := primitive.ToConfig()
		// if config remains the same, return
		// if diff, re-initialize the primitive
		if reflect.DeepEqual(config, curConfig) {
			return nil
		}
		defaultToolset = removeFirstFromToolset(defaultToolset, name)
	}
	_, span := instrumentation.Tracer.Start(
		ctx,
		"toolbox/server/tool/init",
		trace.WithAttributes(attribute.String("tool_type", config.ToolConfigType())),
		trace.WithAttributes(attribute.String("tool_name", name)),
	)
	defer span.End()
	t, err := config.Initialize(r.sources)
	if err != nil {
		return fmt.Errorf("unable to initialize tool %q: %w", name, err)
	}
	r.tools[name] = t
	// add new tool to default toolset
	defaultToolset.Tools = append(defaultToolset.Tools, &t)
	r.toolsets[""] = defaultToolset
	return nil
}

// UpdateToolset creates or update toolset.
func (r *ResourceManager) UpdateToolset(ctx context.Context, name string, config tools.ToolsetConfig, version string) error {
	primitive, exist := r.GetToolset(name)
	instrumentation, err := util.InstrumentationFromContext(ctx)
	if err != nil {
		return err
	}

	if exist {
		curConfig := primitive.ToConfig()
		// if config remains the same, return
		// if diff, re-initialize the primitive
		if reflect.DeepEqual(config, curConfig) {
			return nil
		}
	}

	_, span := instrumentation.Tracer.Start(
		ctx,
		"toolbox/server/toolset/init",
		trace.WithAttributes(attribute.String("toolset.name", name)),
	)
	defer span.End()
	ts, err := config.Initialize(version, r.tools)
	if err != nil {
		return fmt.Errorf("unable to initialize toolset %q: %w", name, err)
	}

	r.toolsets[name] = ts
	return nil
}

// UpdatePrompt creates or update prompt.
func (r *ResourceManager) UpdatePrompt(ctx context.Context, name string, config prompts.PromptConfig) error {
	primitive, exist := r.GetPrompt(name)
	instrumentation, err := util.InstrumentationFromContext(ctx)
	if err != nil {
		return err
	}

	if exist {
		curConfig := primitive.ToConfig()
		// if config remains the same, return
		// if diff, re-initialize the primitive
		if reflect.DeepEqual(config, curConfig) {
			return nil
		}
	}

	_, span := instrumentation.Tracer.Start(
		ctx,
		"toolbox/server/prompt/init",
		trace.WithAttributes(attribute.String("prompt_type", config.PromptConfigType())),
		trace.WithAttributes(attribute.String("prompt_name", name)),
	)
	defer span.End()
	p, err := config.Initialize()
	if err != nil {
		return fmt.Errorf("unable to initialize prompt %q: %w", name, err)
	}
	r.prompts[name] = p
	return nil
}

// DeleteSource deletes a source.
func (r *ResourceManager) DeleteSource(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.sources[name]
	if !ok {
		return false
	}
	delete(r.sources, name)
	return true
}

// DeleteAuthService deletes an auth service.
func (r *ResourceManager) DeleteAuthService(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.authServices[name]
	if !ok {
		return false
	}
	delete(r.authServices, name)
	return true
}

// DeleteEmbeddingModel deletes an embedding model.
func (r *ResourceManager) DeleteEmbeddingModel(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.embeddingModels[name]
	if !ok {
		return false
	}
	delete(r.embeddingModels, name)
	return true
}

// DeleteTool deletes a tool.
func (r *ResourceManager) DeleteTool(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.tools[name]
	if !ok {
		return false
	}
	delete(r.tools, name)

	// Also remove the tool from the default toolset
	// This function do not remove tool from other toolset. User have to update
	// toolset explicitly
	if defaultToolset, toolsetExists := r.toolsets[""]; toolsetExists {
		r.toolsets[""] = removeFirstFromToolset(defaultToolset, name)
	}
	return true
}

// DeleteToolset deletes a toolset.
func (r *ResourceManager) DeleteToolset(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.toolsets[name]
	if !ok {
		return false
	}
	delete(r.toolsets, name)
	return true
}

// DeletePrompt deletes a prompt.
func (r *ResourceManager) DeletePrompt(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.prompts[name]
	if !ok {
		return false
	}
	delete(r.prompts, name)
	return true
}
