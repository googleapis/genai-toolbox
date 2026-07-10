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
	"sync"

	"github.com/googleapis/mcp-toolbox/internal/auth"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/group"
	"github.com/googleapis/mcp-toolbox/internal/prompts"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
)

// ResourceManager contains available resources for the server. Should be initialized with NewResourceManager().
// groups is the source of truth for named collections; the toolset and promptset
// views are derived from it (see GetToolset and GetPromptset).
type ResourceManager struct {
	mu              sync.RWMutex
	sources         map[string]sources.Source
	authServices    map[string]auth.AuthService
	embeddingModels map[string]embeddingmodels.EmbeddingModel
	tools           map[string]tools.Tool
	prompts         map[string]prompts.Prompt
	groups          map[string]group.Group
}

func NewResourceManager(
	sourcesMap map[string]sources.Source,
	authServicesMap map[string]auth.AuthService,
	embeddingModelsMap map[string]embeddingmodels.EmbeddingModel,
	toolsMap map[string]tools.Tool,
	promptsMap map[string]prompts.Prompt,
	groupsMap map[string]group.Group,

) *ResourceManager {
	resourceMgr := &ResourceManager{
		mu:              sync.RWMutex{},
		sources:         sourcesMap,
		authServices:    authServicesMap,
		embeddingModels: embeddingModelsMap,
		tools:           toolsMap,
		prompts:         promptsMap,
		groups:          groupsMap,
	}

	return resourceMgr
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

// GetToolset returns the toolset view derived from the group of the same name.
// The group is the source of truth; the toolset is materialized on demand from
// the group's tool names so callers on the legacy REST path keep a tools.Toolset.
// The manifest's server version is left empty here and set by the caller that
// renders it.
func (r *ResourceManager) GetToolset(toolsetName string) (tools.Toolset, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	g, ok := r.groups[toolsetName]
	if !ok {
		return tools.Toolset{}, false
	}
	toolset, err := tools.ToolsetConfig{Name: g.Name, ToolNames: g.ToolNames}.Initialize("", r.tools)
	if err != nil {
		return tools.Toolset{}, false
	}
	return toolset, true
}

func (r *ResourceManager) GetPrompt(promptName string) (prompts.Prompt, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	prompt, ok := r.prompts[promptName]
	return prompt, ok
}

// GetPromptset returns the promptset view derived from the group of the same
// name. The group is the source of truth; the promptset is materialized on
// demand from the group's prompt names.
func (r *ResourceManager) GetPromptset(promptsetName string) (prompts.Promptset, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	g, ok := r.groups[promptsetName]
	if !ok {
		return prompts.Promptset{}, false
	}
	promptset, err := prompts.PromptsetConfig{Name: g.Name, PromptNames: g.PromptNames}.Initialize("", r.prompts)
	if err != nil {
		return prompts.Promptset{}, false
	}
	return promptset, true
}

// GetGroup returns the group of the given name.
func (r *ResourceManager) GetGroup(groupName string) (group.Group, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	g, ok := r.groups[groupName]
	return g, ok
}

func (r *ResourceManager) SetResources(sourcesMap map[string]sources.Source, authServicesMap map[string]auth.AuthService, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel, toolsMap map[string]tools.Tool, promptsMap map[string]prompts.Prompt, groupsMap map[string]group.Group) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources = sourcesMap
	r.authServices = authServicesMap
	r.embeddingModels = embeddingModelsMap
	r.tools = toolsMap
	r.prompts = promptsMap
	r.groups = groupsMap
}

func (r *ResourceManager) GetSourcesMap() map[string]sources.Source {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copiedMap := make(map[string]sources.Source, len(r.sources))
	for k, v := range r.sources {
		copiedMap[k] = v
	}
	return copiedMap
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

func (r *ResourceManager) GetGroupsMap() map[string]group.Group {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copiedMap := make(map[string]group.Group, len(r.groups))
	for k, v := range r.groups {
		copiedMap[k] = v
	}
	return copiedMap
}
