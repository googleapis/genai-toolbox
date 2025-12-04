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
	"net/http"
	"slices"

	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
	mcputil "github.com/googleapis/genai-toolbox/internal/server/mcp/util"
	v20241105 "github.com/googleapis/genai-toolbox/internal/server/mcp/v20241105"
	v20250326 "github.com/googleapis/genai-toolbox/internal/server/mcp/v20250326"
	v20250618 "github.com/googleapis/genai-toolbox/internal/server/mcp/v20250618"
	"github.com/googleapis/genai-toolbox/internal/server/resources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

// LATEST_PROTOCOL_VERSION is the latest version of the MCP protocol supported.
// Update the version used in InitializeResponse when this value is updated.
const LATEST_PROTOCOL_VERSION = v20250618.PROTOCOL_VERSION

// SUPPORTED_PROTOCOL_VERSIONS is the MCP protocol versions that are supported.
var SUPPORTED_PROTOCOL_VERSIONS = []string{
	v20241105.PROTOCOL_VERSION,
	v20250326.PROTOCOL_VERSION,
	v20250618.PROTOCOL_VERSION,
}

// InitializeResponse runs capability negotiation and protocol version agreement.
// This is the Initialization phase of the lifecycle for MCP client-server connections.
// Always start with the latest protocol version supported.
func InitializeResponse(ctx context.Context, id jsonrpc.RequestId, body []byte, toolboxVersion string) (any, string, error) {
	var req mcputil.InitializeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		err = fmt.Errorf("invalid mcp initialize request: %w", err)
		return jsonrpc.NewError(id, jsonrpc.INVALID_REQUEST, err.Error(), nil), "", err
	}

	var protocolVersion string
	v := req.Params.ProtocolVersion
	if slices.Contains(SUPPORTED_PROTOCOL_VERSIONS, v) {
		protocolVersion = v
	} else {
		protocolVersion = LATEST_PROTOCOL_VERSION
	}

	toolsListChanged := false
	promptsListChanged := false
	result := mcputil.InitializeResult{
		ProtocolVersion: protocolVersion,
		Capabilities: mcputil.ServerCapabilities{
			Tools: &mcputil.ListChanged{
				ListChanged: &toolsListChanged,
			},
			Prompts: &mcputil.ListChanged{
				ListChanged: &promptsListChanged,
			},
		},
		ServerInfo: mcputil.Implementation{
			BaseMetadata: mcputil.BaseMetadata{
				Name: mcputil.SERVER_NAME,
			},
			Version: toolboxVersion,
		},
	}
	res := jsonrpc.JSONRPCResponse{
		Jsonrpc: jsonrpc.JSONRPC_VERSION,
		Id:      id,
		Result:  result,
	}

	return res, protocolVersion, nil
}

// NotificationHandler process notifications request. It MUST NOT send a response.
// Currently Toolbox does not process any notifications.
func NotificationHandler(ctx context.Context, body []byte) error {
	var notification jsonrpc.JSONRPCNotification
	if err := json.Unmarshal(body, &notification); err != nil {
		return fmt.Errorf("invalid notification request: %w", err)
	}
	return nil
}

// ProcessMethod returns a response for the request.
// This is the Operation phase of the lifecycle for MCP client-server connections.
func ProcessMethod(ctx context.Context, mcpVersion string, id jsonrpc.RequestId, method string, toolset tools.Toolset, promptset prompts.Promptset, resourceMgr *resources.ResourceManager, body []byte, header http.Header) (any, error) {
	switch mcpVersion {
	case v20250618.PROTOCOL_VERSION:
		return v20250618.ProcessMethod(ctx, id, method, toolset, promptset, resourceMgr, body, header)
	case v20250326.PROTOCOL_VERSION:
		return v20250326.ProcessMethod(ctx, id, method, toolset, promptset, resourceMgr, body, header)
	default:
		return v20241105.ProcessMethod(ctx, id, method, toolset, promptset, resourceMgr, body, header)
	}
}

// VerifyProtocolVersion verifies if the version string is valid.
func VerifyProtocolVersion(version string) bool {
	return slices.Contains(SUPPORTED_PROTOCOL_VERSIONS, version)
}
