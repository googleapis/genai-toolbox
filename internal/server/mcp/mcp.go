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

	"github.com/googleapis/mcp-toolbox/internal/prompts"
	"github.com/googleapis/mcp-toolbox/internal/server/mcp/jsonrpc"
	mcputil "github.com/googleapis/mcp-toolbox/internal/server/mcp/util"
	v20241105 "github.com/googleapis/mcp-toolbox/internal/server/mcp/v20241105"
	v20250326 "github.com/googleapis/mcp-toolbox/internal/server/mcp/v20250326"
	v20250618 "github.com/googleapis/mcp-toolbox/internal/server/mcp/v20250618"
	v20251125 "github.com/googleapis/mcp-toolbox/internal/server/mcp/v20251125"
	"github.com/googleapis/mcp-toolbox/internal/server/resources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
)

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
	case v20251125.PROTOCOL_VERSION:
		return v20251125.ProcessMethod(ctx, id, method, toolset, promptset, resourceMgr, body, header)
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
	return slices.Contains(mcputil.SUPPORTED_PROTOCOL_VERSIONS, version)
}
