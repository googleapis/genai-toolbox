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
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
	v20251125 "github.com/googleapis/genai-toolbox/internal/server/mcp/v20251125"
)

// NewMCPRequestHeader takes custom headers and appends headers required for MCP.
func NewMCPRequestHeader(t *testing.T, customHeaders map[string]string) map[string]string {
	headers := make(map[string]string)
	for k, v := range customHeaders {
		headers[k] = v
	}
	headers["Content-Type"] = "application/json"
	headers["MCP-Protocol-Version"] = v20251125.PROTOCOL_VERSION
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
func GetMCPToolsList(t *testing.T, requestHeader map[string]string) (int, []any, error) {
	headers := NewMCPRequestHeader(t, requestHeader)

	req := MCPListToolsRequest{
		Jsonrpc: jsonrpc.JSONRPC_VERSION,
		Id:      uuid.New().String(),
		Method:  v20251125.TOOLS_LIST,
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

	resultMap, ok := mcpResp.Result.(map[string]any)
	if !ok {
		t.Fatalf("tools/list result is not a map: %v", mcpResp.Result)
	}

	toolsList, ok := resultMap["tools"].([]any)
	if !ok {
		t.Fatalf("tools/list did not contain tools array: %v", resultMap)
	}

	return resp.StatusCode, toolsList, nil
}

// AssertMCPError asserts that the response contains an error covering the expected message.
func AssertMCPError(t *testing.T, mcpResp *MCPCallToolResponse, wantErrMsg string) {
	t.Helper()
	var errText string
	if mcpResp.Error != nil {
		errText = mcpResp.Error.Message
	} else if mcpResp.Result.IsError {
		for _, content := range mcpResp.Result.Content {
			if content.Type == "text" {
				errText += content.Text
			}
		}
	} else {
		t.Fatalf("expected error containing %q, but got success result: %v", wantErrMsg, mcpResp.Result)
	}

	if !strings.Contains(errText, wantErrMsg) {
		t.Fatalf("expected error text containing %q, got %q", wantErrMsg, errText)
	}
}

// RunMCPToolsListMethod calls tools/list and verifies that the returned tools match the expected list.
func RunMCPToolsListMethod(t *testing.T, expectedOutput []MCPToolManifest) {
	t.Helper()
	statusCodeList, toolsList, errList := GetMCPToolsList(t, nil)
	if errList != nil {
		t.Fatalf("native error executing tools/list: %s", errList)
	}
	if statusCodeList != http.StatusOK {
		t.Fatalf("expected status 200 for tools/list, got %d", statusCodeList)
	}

	// Unmarshal toolsList into []MCPToolManifest
	toolsJSON, err := json.Marshal(toolsList)
	if err != nil {
		t.Fatalf("error marshalling tools list: %v", err)
	}

	var actualTools []MCPToolManifest
	if err := json.Unmarshal(toolsJSON, &actualTools); err != nil {
		t.Fatalf("error unmarshalling tools into MCPToolManifest: %v", err)
	}

	for _, expected := range expectedOutput {
		found := false
		for _, actual := range actualTools {
			if actual.Name == expected.Name {
				found = true
				// Use reflect.DeepEqual to check all fields (description, parameters, etc.)
				if !reflect.DeepEqual(actual, expected) {
					t.Fatalf("tool %s mismatch:\nwant: %+v\ngot: %+v", expected.Name, expected, actual)
				}
				break
			}
		}
		if !found {
			t.Fatalf("tool %s was not found in the tools/list registry", expected.Name)
		}
	}
}

// RunMCPCustomToolCallMethod invokes a tool and compares the result with expected output.
func RunMCPCustomToolCallMethod(t *testing.T, toolName string, arguments map[string]any, want string) {
	t.Helper()
	statusCode, mcpResp, err := InvokeMCPTool(t, toolName, arguments, nil)
	if err != nil {
		t.Fatalf("native error executing %s: %s", toolName, err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusCode)
	}
	if mcpResp.Result.IsError {
		t.Fatalf("%s returned error result: %v", toolName, mcpResp.Result)
	}
	if len(mcpResp.Result.Content) == 0 {
		t.Fatalf("%s returned empty content field", toolName)
	}
	got := mcpResp.Result.Content[0].Text
	if !strings.Contains(got, want) {
		t.Fatalf(`expected %q to contain %q`, got, want)
	}
}
