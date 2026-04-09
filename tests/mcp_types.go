// Copyright 2026 Google LLC
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

package tests

import (
	"github.com/googleapis/mcp-toolbox/internal/server/mcp/jsonrpc"
	v20251125 "github.com/googleapis/mcp-toolbox/internal/server/mcp/v20251125"
)

// CallToolParams represents the internal payload of an MCP tool call request
type CallToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// MCPListToolsRequest encapsulates the standard JSON-RPC request format targeting tools/list
type MCPListToolsRequest struct {
	Jsonrpc string            `json:"jsonrpc"`
	Id      jsonrpc.RequestId `json:"id"`
	Method  string            `json:"method"`
}

// MCPCallToolRequest encapsulates the standard JSON-RPC request format targeting tools/call
type MCPCallToolRequest struct {
	Jsonrpc string            `json:"jsonrpc"`
	Id      jsonrpc.RequestId `json:"id"`
	Method  string            `json:"method"`
	Params  CallToolParams    `json:"params"`
}

// MCPCallToolResponse provides a strongly-typed unmarshal target for MCP tool call results,
// bypassing the generic interface{} Result used in the standard jsonrpc.JSONRPCResponse.
type MCPCallToolResponse struct {
	Jsonrpc string                   `json:"jsonrpc"`
	Id      jsonrpc.RequestId        `json:"id"`
	Result  v20251125.CallToolResult `json:"result,omitempty"`
	Error   *jsonrpc.Error           `json:"error,omitempty"`
}

// NewMCPCallToolRequest is a helper to quickly generate a standard jsonrpc request payload.
func NewMCPCallToolRequest(id jsonrpc.RequestId, toolName string, args map[string]any) MCPCallToolRequest {
	return MCPCallToolRequest{
		Jsonrpc: jsonrpc.JSONRPC_VERSION,
		Id:      id,
		Method:  v20251125.TOOLS_CALL,
		Params: CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	}
}

// MCPListToolsResponse is a strongly-typed unmarshal target for tools/list results
type MCPListToolsResponse struct {
	Jsonrpc string            `json:"jsonrpc"`
	Id      jsonrpc.RequestId `json:"id"`
	Result  struct {
		Tools []MCPToolManifest `json:"tools"`
	} `json:"result,omitempty"`
	Error *jsonrpc.Error `json:"error,omitempty"`
}

// MCPToolManifest is a copy of tools.McpManifest used for integration testing purposes
type MCPToolManifest struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}
