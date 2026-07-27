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

package primitives

import (
	"sync"

	"github.com/googleapis/mcp-toolbox/internal/auth"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/group"
	"github.com/googleapis/mcp-toolbox/internal/prompts"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
)

// PrimitiveManager contains available resources for the server. Should be initialized with NewPrimitiveManager().
// groups is the source of truth for named collections; toolset views (manifests)
// are derived from the group on demand by the callers that render them.
type PrimitiveManager struct {
	mu              sync.RWMutex
	sources         map[string]sources.Source
	authServices    map[string]auth.AuthService
	embeddingModels map[string]embeddingmodels.EmbeddingModel
	tools           map[string]tools.Tool
	prompts         map[string]prompts.Prompt
	groups          map[string]group.Group
}

func NewPrimitiveManager(
	sourcesMap map[string]sources.Source,
	authServicesMap map[string]auth.AuthService,
	embeddingModelsMap map[string]embeddingmodels.EmbeddingModel,
	toolsMap map[string]tools.Tool,
	promptsMap map[string]prompts.Prompt,
	groupsMap map[string]group.Group,

) *PrimitiveManager {
	primitiveMgr := &PrimitiveManager{
		mu:              sync.RWMutex{},
		sources:         sourcesMap,
		authServices:    authServicesMap,
		embeddingModels: embeddingModelsMap,
		tools:           toolsMap,
		prompts:         promptsMap,
		groups:          groupsMap,
	}

	return primitiveMgr
}

func (r *PrimitiveManager) GetSource(sourceName string) (sources.Source, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	source, ok := r.sources[sourceName]
	return source, ok
}

func (r *PrimitiveManager) GetAuthService(authServiceName string) (auth.AuthService, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	authService, ok := r.authServices[authServiceName]
	return authService, ok
}

func (r *PrimitiveManager) GetEmbeddingModel(embeddingModelName string) (embeddingmodels.EmbeddingModel, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	model, ok := r.embeddingModels[embeddingModelName]
	return model, ok
}

func (r *PrimitiveManager) GetTool(toolName string) (tools.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[toolName]
	return tool, ok
}

func (r *PrimitiveManager) GetPrompt(promptName string) (prompts.Prompt, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	prompt, ok := r.prompts[promptName]
	return prompt, ok
}

// GetGroup returns the group of the given name.
func (r *PrimitiveManager) GetGroup(groupName string) (group.Group, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	g, ok := r.groups[groupName]
	return g, ok
}

func (r *PrimitiveManager) SetPrimitives(sourcesMap map[string]sources.Source, authServicesMap map[string]auth.AuthService, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel, toolsMap map[string]tools.Tool, promptsMap map[string]prompts.Prompt, groupsMap map[string]group.Group) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources = sourcesMap
	r.authServices = authServicesMap
	r.embeddingModels = embeddingModelsMap
	r.tools = toolsMap
	r.prompts = promptsMap
	r.groups = groupsMap
}

func (r *PrimitiveManager) GetAuthServiceMap() map[string]auth.AuthService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copiedMap := make(map[string]auth.AuthService, len(r.authServices))
	for k, v := range r.authServices {
		copiedMap[k] = v
	}
	return copiedMap
}

func (r *PrimitiveManager) GetEmbeddingModelMap() map[string]embeddingmodels.EmbeddingModel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copiedMap := make(map[string]embeddingmodels.EmbeddingModel, len(r.embeddingModels))
	for k, v := range r.embeddingModels {
		copiedMap[k] = v
	}
	return copiedMap
}

func (r *PrimitiveManager) GetPromptsMap() map[string]prompts.Prompt {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copiedMap := make(map[string]prompts.Prompt, len(r.prompts))
	for k, v := range r.prompts {
		copiedMap[k] = v
	}
	return copiedMap
}

func (r *PrimitiveManager) GetGroupsMap() map[string]group.Group {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copiedMap := make(map[string]group.Group, len(r.groups))
	for k, v := range r.groups {
		copiedMap[k] = v
	}
	return copiedMap
}
