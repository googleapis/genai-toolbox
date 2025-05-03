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
	v20241105 "github.com/googleapis/genai-toolbox/internal/server/mcp/v20241105"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

// JSONRPC_VERSION is the version of JSON-RPC used by MCP.
const JSONRPC_VERSION = "2.0"

// LATEST_PROTOCOL_VERSION is the latest version of the MCP protocol supported.
const LATEST_PROTOCOL_VERSION = v20241105.PROTOCOL_VERSION

// SUPPORTED_PROTOCOL_VERSION is the MCP protocol versions that are supported.
var SUPPORTED_PROTOCOL_VERSION = []string{v20241105.PROTOCOL_VERSION}

// Standard JSON-RPC error codes
const (
	PARSE_ERROR      = -32700
	INVALID_REQUEST  = -32600
	METHOD_NOT_FOUND = -32601
	INVALID_PARAMS   = -32602
	INTERNAL_ERROR   = -32603
)

// NewInitializeResponse return a response for the initialize method.
func NewInitializeResponse(mcpVersion string, id any, body []byte, toolboxVersion string) any {
	switch mcpVersion {
	case v20241105.PROTOCOL_VERSION:
		return v20241105.Initialize(id, body, toolboxVersion)
	default:
		return nil
	}
}

// GetToolsListParam unmarshals the tools/list request.
func GetToolsListParam(mcpVersion string, id any, body []byte) any {
	switch mcpVersion {
	case v20241105.PROTOCOL_VERSION:
		return v20241105.GetToolsListParam(id, body)
	default:
		return nil
	}
}

// NewToolsListResponse return a response for the tools/list method.
func NewToolsListResponse(mcpVersion string, id any, mcpManifest []tools.McpManifest) any {
	switch mcpVersion {
	case v20241105.PROTOCOL_VERSION:
		return v20241105.ToolsList(id, mcpManifest)
	default:
		return nil
	}
}

// GetToolParam returns tool name and tool argument from the tools/call request.
func GetToolParam(mcpVersion string, id any, body []byte) (string, map[string]any, any) {
	switch mcpVersion {
	case v20241105.PROTOCOL_VERSION:
		return v20241105.GetToolParam(id, body)
	default:
		return "", nil, nil
	}
}

// NewToolsCallResponse return a response for the tools/call method.
func NewToolsCallResponse(mcpVersion string, id any, results []any, err error) any {
	switch mcpVersion {
	case v20241105.PROTOCOL_VERSION:
		return v20241105.ToolsCall(id, results, err)
	default:
		return nil
	}
}

// ProcessNotification process the notifications received.
func ProcessNotifications(body []byte, mcpVersion string) any {
	switch mcpVersion {
	case v20241105.PROTOCOL_VERSION:
		if err := v20241105.ProcessNotifications(body); err != nil {
			return err
		}
	default:
		return nil
	}
	return nil
}

// NewJSONRPCError return a valid JSONRPCError based on the protocol version.
func NewJSONRPCError(mcpVersion string, id any, code int, message string, data any) any {
	switch mcpVersion {
	case v20241105.PROTOCOL_VERSION:
		return v20241105.NewJSONRPCError(id, code, message, data)
	default:
		return nil
	}
}
