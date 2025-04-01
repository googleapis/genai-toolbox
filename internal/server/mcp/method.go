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

package mcp

import (
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func Initialize(version string) InitializeResult {
	toolsListChanged := false
	result := InitializeResult{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		Capabilities: ServerCapabilities{
			Tools: &ListChanged{
				ListChanged: &toolsListChanged,
			},
		},
		ServerInfo: Implementation{
			Name:    SERVER_NAME,
			Version: version,
		},
	}
	return result
}

// ToolsList return a ListToolsResult
func ToolsList(toolset tools.Toolset) ListToolsResult {
	mcpManifest := toolset.McpManifest

	result := ListToolsResult{
		Tools: mcpManifest,
	}
	return result
}
