// Copyright 2024 Google LLC
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
	"slices"

	"github.com/googleapis/genai-toolbox/internal/sources"
)

type ToolConfig interface {
	ToolConfigKind() string
	Initialize(map[string]sources.Source) (Tool, error)
}

type Tool interface {
	Invoke(ParamValues) ([]any, error)
	ParseParams(map[string]any, map[string]map[string]any) (ParamValues, error)
	Manifest() Manifest
	Authorized([]string) bool
}

// Manifest is the representation of tools sent to Client SDKs.
type Manifest struct {
	Description string              `json:"description"`
	Parameters  []ParameterManifest `json:"parameters"`
}

// Helper function that returns if a tool invocation request is authorized
func IsAuthorized(authRequiredSources []string, verifiedAuthSources []string) bool {
	if len(authRequiredSources) == 0 {
		// no authorization requirement
		return true
	}
	for _, a := range authRequiredSources {
		if slices.Contains(verifiedAuthSources, a) {
			return true
		}
	}
	return false
}
