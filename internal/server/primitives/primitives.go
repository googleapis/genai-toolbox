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
	"strings"
	"sync"

	"github.com/googleapis/mcp-toolbox/internal/auth"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/prompts"
	"github.com/googleapis/mcp-toolbox/internal/resources"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
)

// PrimitiveManager contains available primitives for the server. Should be initialized with NewPrimitiveManager().
type PrimitiveManager struct {
	mu                sync.RWMutex
	sources           map[string]sources.Source
	authServices      map[string]auth.AuthService
	embeddingModels   map[string]embeddingmodels.EmbeddingModel
	tools             map[string]tools.Tool
	toolsets          map[string]tools.Toolset
	prompts           map[string]prompts.Prompt
	promptsets        map[string]prompts.Promptset
	resources         map[string]resources.Resource
	resourceTemplates map[string]resources.Resource
}

func NewPrimitiveManager(
	sourcesMap map[string]sources.Source,
	authServicesMap map[string]auth.AuthService,
	embeddingModelsMap map[string]embeddingmodels.EmbeddingModel,
	toolsMap map[string]tools.Tool, toolsetsMap map[string]tools.Toolset,
	promptsMap map[string]prompts.Prompt, promptsetsMap map[string]prompts.Promptset,
	resourcesMap map[string]resources.Resource,
	resourceTemplatesMap map[string]resources.Resource,
) *PrimitiveManager {
	primitiveMgr := &PrimitiveManager{
		mu:                sync.RWMutex{},
		sources:           sourcesMap,
		authServices:      authServicesMap,
		embeddingModels:   embeddingModelsMap,
		tools:             toolsMap,
		toolsets:          toolsetsMap,
		prompts:           promptsMap,
		promptsets:        promptsetsMap,
		resources:         resourcesMap,
		resourceTemplates: resourceTemplatesMap,
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

func (r *PrimitiveManager) GetToolset(toolsetName string) (tools.Toolset, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	toolset, ok := r.toolsets[toolsetName]
	return toolset, ok
}

func (r *PrimitiveManager) GetPrompt(promptName string) (prompts.Prompt, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	prompt, ok := r.prompts[promptName]
	return prompt, ok
}

func (r *PrimitiveManager) GetPromptset(promptsetName string) (prompts.Promptset, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	promptset, ok := r.promptsets[promptsetName]
	return promptset, ok
}

func (r *PrimitiveManager) GetResource(name string) (resources.Resource, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res, ok := r.resources[name]
	return res, ok
}

func (r *PrimitiveManager) GetResourceSet(name string) (resources.ResourceSet, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var uris []string
	for _, res := range r.resources {
		uris = append(uris, res.ResourceURI())
	}
	for _, res := range r.resourceTemplates {
		uris = append(uris, res.ResourceURI())
	}

	return resources.ResourceSet{
		Name:         name,
		ResourceURIs: uris,
	}, true
}

func (r *PrimitiveManager) GetResourceTemplate(name string) (resources.Resource, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res, ok := r.resourceTemplates[name]
	return res, ok
}

func (r *PrimitiveManager) ListResources() []resources.Resource {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []resources.Resource
	for _, res := range r.resources {
		list = append(list, res)
	}
	return list
}

func (r *PrimitiveManager) ListResourceTemplates() []resources.Resource {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []resources.Resource
	for _, res := range r.resourceTemplates {
		list = append(list, res)
	}
	return list
}

func (r *PrimitiveManager) ResolveResource(uri string) (resources.Resource, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. Search static resources
	for _, res := range r.resources {
		if res.ResourceURI() == uri {
			return res, true
		}
	}

	// 2. Search resource templates
	for _, res := range r.resourceTemplates {
		templateURI := res.ResourceURI()
		if strings.Contains(templateURI, "{path}") {
			prefix := strings.Split(templateURI, "{path}")[0]
			if strings.HasPrefix(uri, prefix) {
				return res, true
			}
		}
	}

	return nil, false
}

func (r *PrimitiveManager) SetPrimitives(
	sourcesMap map[string]sources.Source,
	authServicesMap map[string]auth.AuthService,
	embeddingModelsMap map[string]embeddingmodels.EmbeddingModel,
	toolsMap map[string]tools.Tool,
	toolsetsMap map[string]tools.Toolset,
	promptsMap map[string]prompts.Prompt,
	promptsetsMap map[string]prompts.Promptset,
	resourcesMap map[string]resources.Resource,
	resourceTemplatesMap map[string]resources.Resource,
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources = sourcesMap
	r.authServices = authServicesMap
	r.embeddingModels = embeddingModelsMap
	r.tools = toolsMap
	r.toolsets = toolsetsMap
	r.prompts = promptsMap
	r.promptsets = promptsetsMap
	r.resources = resourcesMap
	r.resourceTemplates = resourceTemplatesMap
}


func (r *PrimitiveManager) GetSourcesMap() map[string]sources.Source {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copiedMap := make(map[string]sources.Source, len(r.sources))
	for k, v := range r.sources {
		copiedMap[k] = v
	}
	return copiedMap
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

func (r *PrimitiveManager) GetToolsMap() map[string]tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copiedMap := make(map[string]tools.Tool, len(r.tools))
	for k, v := range r.tools {
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

func (r *PrimitiveManager) GetToolsetsMap() map[string]tools.Toolset {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copiedMap := make(map[string]tools.Toolset, len(r.toolsets))
	for k, v := range r.toolsets {
		copiedMap[k] = v
	}
	return copiedMap
}

func (r *PrimitiveManager) GetPromptsetsMap() map[string]prompts.Promptset {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copiedMap := make(map[string]prompts.Promptset, len(r.promptsets))
	for k, v := range r.promptsets {
		copiedMap[k] = v
	}
	return copiedMap
}
