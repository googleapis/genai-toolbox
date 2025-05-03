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

package v20241105

import (
	"encoding/json"
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/tools"
)

// Initialize returns a response for the initialize method
func Initialize(id any, body []byte, toolboxVersion string) any {
	var req InitializeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		err = fmt.Errorf("invalid mcp initialize request: %w", err)
		return NewJSONRPCError(id, INVALID_REQUEST, err.Error(), nil)
	}

	toolsListChanged := false
	result := InitializeResult{
		ProtocolVersion: PROTOCOL_VERSION,
		Capabilities: ServerCapabilities{
			Tools: &ListChanged{
				ListChanged: &toolsListChanged,
			},
		},
		ServerInfo: Implementation{
			Name:    SERVER_NAME,
			Version: toolboxVersion,
		},
	}

	return JSONRPCResponse{
		Jsonrpc: JSONRPC_VERSION,
		Id:      id,
		Result:  result,
	}
}

// GetToolsListParam returns error (if any) during the unmarshalling process
func GetToolsListParam(id any, body []byte) any {
	var req ListToolsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		err = fmt.Errorf("invalid mcp tools list request: %w", err)
		return NewJSONRPCError(id, INVALID_REQUEST, err.Error(), nil)
	}
	return nil
}

// ToolsList return a response with ListToolsResult
func ToolsList(id any, mcpManifest []tools.McpManifest) any {
	result := ListToolsResult{
		Tools: mcpManifest,
	}
	return JSONRPCResponse{
		Jsonrpc: JSONRPC_VERSION,
		Id:      id,
		Result:  result,
	}
}

// GetToolParam retrieves tool name and tool argument from the request
func GetToolParam(id any, body []byte) (string, map[string]any, any) {
	var req CallToolRequest
	if err := json.Unmarshal(body, &req); err != nil {
		err = fmt.Errorf("invalid mcp tools call request: %w", err)
		return "", nil, NewJSONRPCError(id, INVALID_REQUEST, err.Error(), nil)
	}
	toolName := req.Params.Name
	toolArgument := req.Params.Arguments
	return toolName, toolArgument, nil
}

// ToolsCall runs tool invocation and return a CallToolResult
func ToolsCall(id any, res []any, err error) any {
	if err != nil {
		text := TextContent{
			Type: "text",
			Text: err.Error(),
		}
		return JSONRPCResponse{
			Jsonrpc: JSONRPC_VERSION,
			Id:      id,
			Result:  CallToolResult{Content: []TextContent{text}, IsError: true},
		}
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
	return JSONRPCResponse{
		Jsonrpc: JSONRPC_VERSION,
		Id:      id,
		Result:  CallToolResult{Content: content},
	}
}

// ProcessNotifications process the notifications received.
// Toolbox currently doesn't support any notifications.
func ProcessNotifications(body []byte) any {
	var notification JSONRPCNotification
	if err := json.Unmarshal(body, &notification); err != nil {
		err = fmt.Errorf("invalid notification request: %w", err)
		return NewJSONRPCError("", PARSE_ERROR, err.Error(), nil)
	}
	return nil
}

// NewJSONRPCError is the response sent back when an error has been encountered in mcp.
func NewJSONRPCError(id any, code int, message string, data any) JSONRPCError {
	return JSONRPCError{
		Jsonrpc: JSONRPC_VERSION,
		Id:      id,
		Error: McpError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}
