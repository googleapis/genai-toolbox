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
	"reflect"
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

type ResourceKind string

const (
	KindSource         ResourceKind = "source"
	KindAuthService    ResourceKind = "authservice"
	KindEmbeddingModel ResourceKind = "embeddingmodel"
	KindTool           ResourceKind = "tool"
	KindToolset        ResourceKind = "toolset"
	KindPrompt         ResourceKind = "prompt"
	KindPromptset      ResourceKind = "promptset"
)

// ResourceManager contains available resources for the server. Should be initialized with NewResourceManager().
type ResourceManager struct {
	mu              sync.RWMutex
	serverVersion   string
	sources         map[string]sources.Source
	authServices    map[string]auth.AuthService
	embeddingModels map[string]embeddingmodels.EmbeddingModel
	tools           map[string]tools.Tool
	toolsets        map[string]tools.Toolset
	prompts         map[string]prompts.Prompt
	promptsets      map[string]prompts.Promptset
}

func NewResourceManager(
	serverVersion string,
	sourcesMap map[string]sources.Source,
	authServicesMap map[string]auth.AuthService,
	embeddingModelsMap map[string]embeddingmodels.EmbeddingModel,
	toolsMap map[string]tools.Tool, toolsetsMap map[string]tools.Toolset,
	promptsMap map[string]prompts.Prompt, promptsetsMap map[string]prompts.Promptset,

) *ResourceManager {
	resourceMgr := &ResourceManager{
		mu:              sync.RWMutex{},
		serverVersion:   serverVersion,
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
	r.mu.Lock()
	defer r.mu.Unlock()
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
	r.mu.Lock()
	defer r.mu.Unlock()
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
	r.mu.Lock()
	defer r.mu.Unlock()
	r.embeddingModels[name] = em
	return nil
}

// UpdateTool creates or update tool.
func (r *ResourceManager) UpdateTool(ctx context.Context, name string, config tools.ToolConfig) error {
	primitive, exist := r.GetTool(name)
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
		"toolbox/server/tool/init",
		trace.WithAttributes(attribute.String("tool_type", config.ToolConfigType())),
		trace.WithAttributes(attribute.String("tool_name", name)),
	)
	defer span.End()
	t, err := config.Initialize(r.sources)
	if err != nil {
		return fmt.Errorf("unable to initialize tool %q: %w", name, err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	// update the default toolset - this will update list manifests
	defaultToolset := r.toolsets[""]
	// only need to append toolName if tool does not currently exist
	if !exist {
		defaultToolset.ToolNames = append(defaultToolset.ToolNames, name)
	}
	r.tools[name] = t
	err = defaultToolset.Initialize(r.serverVersion, r.tools)
	if err != nil {
		delete(r.tools, name)
		return fmt.Errorf("error updating toolset: %w", err)
	}
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
	r.mu.Lock()
	defer r.mu.Unlock()
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
	r.mu.Lock()
	defer r.mu.Unlock()

	// update the default promptset - this will update list manifests
	defaultPromptset := r.promptsets[""]
	// only need to append promptNames if prompt does not currently exist
	if !exist {
		defaultPromptset.PromptNames = append(defaultPromptset.PromptNames, name)
	}
	r.prompts[name] = p
	err = defaultPromptset.Initialize(r.serverVersion, r.prompts)
	if err != nil {
		delete(r.prompts, name)
		return fmt.Errorf("error updating promptset: %w", err)
	}
	r.promptsets[""] = defaultPromptset
	return nil
}
