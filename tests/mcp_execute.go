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

func ExecuteMCPToolCall(t *testing.T, toolName string, arguments map[string]any, requestHeader map[string]string) (string, error) {
	if requestHeader == nil {
		requestHeader = make(map[string]string)
	}
	requestHeader["Content-Type"] = "application/json"
	if requestHeader["Mcp-Session-Id"] == "" {
		requestHeader["Mcp-Session-Id"] = RunInitialize(t, v20250618.PROTOCOL_VERSION)
	}

	req := NewMCPCallToolRequest("1", toolName, arguments)
	reqBody, _ := json.Marshal(req)

	resp, respBody := RunRequest(t, http.MethodPost, "http://127.0.0.1:5000/mcp", bytes.NewBuffer(reqBody), requestHeader)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("response status code is not 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var mcpResp MCPCallToolResponse
	if err := json.Unmarshal(respBody, &mcpResp); err != nil {
		t.Fatalf("error parsing mcp response body: %v\nraw body: %s", err, string(respBody))
	}
	if mcpResp.Error != nil {
		return "", fmt.Errorf("MCP error %d: %s", mcpResp.Error.Code, mcpResp.Error.Message)
	}
	if len(mcpResp.Result.Content) == 0 {
		return "null", nil
	}

	var contentText string
	for _, c := range mcpResp.Result.Content {
		if c.Type == "text" {
			contentText += c.Text
		}
	}
	return strings.TrimSpace(contentText), nil
}
