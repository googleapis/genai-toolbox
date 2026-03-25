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
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
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

// InvokeMCPTool is a transparent, native JSON-RPC execution harness for tests.
func InvokeMCPTool(t *testing.T, toolName string, arguments map[string]any, requestHeader map[string]string) (int, *MCPCallToolResponse, error) {
	headers := NewMCPRequestHeader(t, requestHeader)

	req := NewMCPCallToolRequest(uuid.New().String(), toolName, arguments)
	reqBody, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("error marshalling request body: %v", err)
	}

	resp, respBody := RunRequest(t, http.MethodPost, "http://127.0.0.1:5000/mcp", bytes.NewBuffer(reqBody), headers)

	var mcpResp MCPCallToolResponse
	if err := json.Unmarshal(respBody, &mcpResp); err != nil {
		if resp.StatusCode != http.StatusOK {
			return resp.StatusCode, nil, fmt.Errorf("%s", string(respBody))
		}
		t.Fatalf("error parsing mcp response body: %v\nraw body: %s", err, string(respBody))
	}

	return resp.StatusCode, &mcpResp, nil
}

// GetMCPToolsList is a JSON-RPC harness that fetches the tools/list registry.
func GetMCPToolsList(t *testing.T, requestHeader map[string]string) (int, *jsonrpc.JSONRPCResponse, error) {
	headers := NewMCPRequestHeader(t, requestHeader)

	req := MCPListToolsRequest{
		Jsonrpc: jsonrpc.JSONRPC_VERSION,
		Id:      uuid.New().String(),
		Method:  v20250618.TOOLS_LIST,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("error marshalling tools/list request body: %v", err)
	}

	resp, respBody := RunRequest(t, http.MethodPost, "http://127.0.0.1:5000/mcp", bytes.NewBuffer(reqBody), headers)

	var mcpResp jsonrpc.JSONRPCResponse
	if err := json.Unmarshal(respBody, &mcpResp); err != nil {
		if resp.StatusCode != http.StatusOK {
			return resp.StatusCode, nil, fmt.Errorf("%s", string(respBody))
		}
		t.Fatalf("error parsing tools/list response: %v\nraw body: %s", err, string(respBody))
	}

	return resp.StatusCode, &mcpResp, nil
}

// ExecuteMCPToolCall is a helper function to send HTTP requests to MCP endpoint and return the response
func ExecuteMCPToolCall(t *testing.T, toolName string, arguments map[string]any, requestHeader map[string]string) (int, string, error) {
	headers := NewMCPRequestHeader(t, requestHeader)

	req := NewMCPCallToolRequest(uuid.New().String(), toolName, arguments)
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
		return resp.StatusCode, "", fmt.Errorf("%s", mcpResp.Error.Message)
	}

	if mcpResp.Result.IsError {
		// If it's an application-level MCP tool failure, map it as an error text
		var errText string
		for _, c := range mcpResp.Result.Content {
			if c.Type == "text" {
				errText += c.Text
			}
		}
		return resp.StatusCode, strings.TrimSpace(errText), nil
	}
	if len(mcpResp.Result.Content) == 0 {
		return resp.StatusCode, "null", nil
	}

	var textBlocks []string
	for _, c := range mcpResp.Result.Content {
		if c.Type == "text" {
			textBlocks = append(textBlocks, strings.TrimSpace(c.Text))
		}
	}

	if len(textBlocks) == 0 {
		return resp.StatusCode, "null", nil
	}
	if len(textBlocks) == 1 {
		return resp.StatusCode, textBlocks[0], nil
	}

	// For legacy assertions: if multiple blocks are returned and they look like JSON, wrap them into a JSON array
	first := textBlocks[0]
	if strings.HasPrefix(first, "{") || strings.HasPrefix(first, "[") || strings.HasPrefix(first, "\"") {
		return resp.StatusCode, "[" + strings.Join(textBlocks, ",") + "]", nil
	}

	return resp.StatusCode, strings.Join(textBlocks, "\n"), nil
}

// InterceptLegacyDo intercepts a hardcoded HTTP request destined for /api/tool/.../invoke,
// dynamically converts it into a local ExecuteMCPToolCall, and wraps the
// MCP string (or JSON-RPC logic error) inside a standard Go *http.Response.
func InterceptLegacyDo(t *testing.T, req *http.Request) (*http.Response, error) {
	// If the request is natively meant for the modern /mcp endpoint, pass it through directly!
	if strings.HasPrefix(req.URL.Path, "/mcp") {
		return http.DefaultClient.Do(req)
	}

	pathParts := strings.Split(req.URL.Path, "/")
	// e.g., /api/tool/cloud-gda-query/invoke -> length 5, tool is pathParts[3]
	if len(pathParts) < 4 || pathParts[2] != "tool" {
		t.Fatalf("InterceptLegacyDo: invalid or unsupported legacy URL path %s", req.URL.Path)
	}
	toolName := pathParts[3]

	var reqBodyBytes []byte
	var err error
	if req.Body != nil {
		reqBodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("InterceptLegacyDo: failed to read body: %v", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(reqBodyBytes))
	}

	var args map[string]any
	if len(reqBodyBytes) > 0 {
		if err := json.Unmarshal(reqBodyBytes, &args); err != nil {
			t.Fatalf("InterceptLegacyDo: failed to unmarshal body to map[string]any: %v", err)
		}
	}

	headers := map[string]string{}
	for k, v := range req.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	statusCode, resultStr, err := ExecuteMCPToolCall(t, toolName, args, headers)

	var mockPayload []byte
	if err != nil {
		mockPayload = []byte(fmt.Sprintf(`{"error":%q}`, err.Error()))
	} else if statusCode != http.StatusOK {
		mockPayload = []byte(fmt.Sprintf(`{"error":%q}`, resultStr))
	} else {
		mockPayload = []byte(fmt.Sprintf(`{"result":%q}`, resultStr))
	}

	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader(mockPayload)),
	}, nil
}
