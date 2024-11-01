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
	"github.com/googleapis/genai-toolbox/internal/sources"
)

type Config interface {
	ToolKind() string
	Initialize(map[string]sources.Source) (Tool, error)
}

type Tool interface {
	Invoke([]any) (string, error)
	ParseParams(data map[string]any) ([]any, error)
	Manifest() Manifest
}

// Manifest is the representation of tools sent to Client SDKs.
type Manifest struct {
	Description string              `json:"description"`
	Parameters  []ParameterManifest `json:"parameters"`
}
