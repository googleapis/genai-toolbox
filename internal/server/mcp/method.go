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
	"context"
	"encoding/json"
	"fmt"

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

// ToolCall runs tool invocation and return a CallToolResult
func ToolCall(ctx context.Context, tool tools.Tool, params tools.ParamValues) CallToolResult {
	res, err := tool.Invoke(ctx, params)
	if err != nil {
		text := TextContent{
			Type: "text",
			Text: err.Error(),
		}
		return CallToolResult{Content: []TextContent{text}, IsError: true}
	}

	content := make([]TextContent, 0)
	for _, d := range res {
		text := TextContent{Type: "text"}
		dM, err := json.Marshal(d)
		if err != nil {
			text.Text = fmt.Sprintf("fail to marshal: %s, result: %s", err, d)
		} else {
			text.Text = string(dM)
		}
		content = append(content, text)
	}
	return CallToolResult{Content: content}
}
