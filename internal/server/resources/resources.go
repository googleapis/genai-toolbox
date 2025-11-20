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

	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

// ResourceManager contains available resources for the server. Should be initialized with NewResourceManager().
type ResourceManager struct {
	mu           sync.RWMutex
	sources      map[string]sources.Source
	authServices map[string]auth.AuthService
	tools        map[string]tools.Tool
	toolsets     map[string]tools.Toolset
	prompts      map[string]prompts.Prompt
	promptsets   map[string]prompts.Promptset
}

func NewResourceManager(
	sourcesMap map[string]sources.Source,
	authServicesMap map[string]auth.AuthService,
	toolsMap map[string]tools.Tool, toolsetsMap map[string]tools.Toolset,
	promptsMap map[string]prompts.Prompt, promptsetsMap map[string]prompts.Promptset,

) *ResourceManager {
	resourceMgr := &ResourceManager{
		mu:           sync.RWMutex{},
		sources:      sourcesMap,
		authServices: authServicesMap,
		tools:        toolsMap,
		toolsets:     toolsetsMap,
		prompts:      promptsMap,
		promptsets:   promptsetsMap,
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

func (r *ResourceManager) SetResources(sourcesMap map[string]sources.Source, authServicesMap map[string]auth.AuthService, toolsMap map[string]tools.Tool, toolsetsMap map[string]tools.Toolset, promptsMap map[string]prompts.Prompt, promptsetsMap map[string]prompts.Promptset) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources = sourcesMap
	r.authServices = authServicesMap
	r.tools = toolsMap
	r.toolsets = toolsetsMap
	r.prompts = promptsMap
	r.promptsets = promptsetsMap
}

func (r *ResourceManager) GetAuthServiceMap() map[string]auth.AuthService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.authServices
}

func (r *ResourceManager) GetToolsMap() map[string]tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools
}

func (r *ResourceManager) GetPromptsMap() map[string]prompts.Prompt {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.prompts
}
