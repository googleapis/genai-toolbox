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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	v20250618 "github.com/googleapis/genai-toolbox/internal/server/mcp/v20250618"
)

// NewMCPRequestHeader takes custom headers and append headers required for MCP
func NewMCPRequestHeader(t *testing.T, customHeaders map[string]string) map[string]string {
	headers := make(map[string]string)
	for k, v := range customHeaders {
		headers[k] = v
	}
	headers["Content-Type"] = "application/json"
	headers["MCP-Protocol-Version"] = v20250618.PROTOCOL_VERSION
	return headers
}

// ExecuteMCPToolCall is a helper function to send HTTP requests to MCP endpoint and return the response
func ExecuteMCPToolCall(t *testing.T, toolName string, arguments map[string]any, requestHeader map[string]string) (int, string, error) {
	headers := NewMCPRequestHeader(t, requestHeader)

	req := NewMCPCallToolRequest("1", toolName, arguments)
	reqBody, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("error marshalling request body: %v", err)
	}

	resp, respBody := RunRequest(t, http.MethodPost, "http://127.0.0.1:5000/mcp", bytes.NewBuffer(reqBody), headers)

	var mcpResp MCPCallToolResponse
	if err := json.Unmarshal(respBody, &mcpResp); err != nil {
		// If unmarshal fails on error HTTP code, bubble the exact string payload as error rather than crashing
		if resp.StatusCode != http.StatusOK {
			return resp.StatusCode, "", fmt.Errorf("%s", string(respBody))
		}
		t.Fatalf("error parsing mcp response body: %v\nraw body: %s", err, string(respBody))
	}
	if mcpResp.Error != nil {
		return resp.StatusCode, "", fmt.Errorf("MCP error %d: %s", mcpResp.Error.Code, mcpResp.Error.Message)
	}
	if len(mcpResp.Result.Content) == 0 {
		return resp.StatusCode, "null", nil
	}

	var contentText string
	for _, c := range mcpResp.Result.Content {
		if c.Type == "text" {
			contentText += c.Text
		}
	}
	return resp.StatusCode, strings.TrimSpace(contentText), nil
}
