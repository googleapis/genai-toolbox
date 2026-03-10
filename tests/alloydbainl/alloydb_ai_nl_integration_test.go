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

package alloydbainl

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	AlloyDBAINLSourceType = "alloydb-postgres"
	AlloyDBAINLToolType   = "alloydb-ai-nl"
	AlloyDBAINLProject    = os.Getenv("ALLOYDB_AI_NL_PROJECT")
	AlloyDBAINLRegion     = os.Getenv("ALLOYDB_AI_NL_REGION")
	AlloyDBAINLCluster    = os.Getenv("ALLOYDB_AI_NL_CLUSTER")
	AlloyDBAINLInstance   = os.Getenv("ALLOYDB_AI_NL_INSTANCE")
	AlloyDBAINLDatabase   = os.Getenv("ALLOYDB_AI_NL_DATABASE")
	AlloyDBAINLUser       = os.Getenv("ALLOYDB_AI_NL_USER")
	AlloyDBAINLPass       = os.Getenv("ALLOYDB_AI_NL_PASS")
)

func getAlloyDBAINLVars(t *testing.T) map[string]any {
	switch "" {
	case AlloyDBAINLProject:
		t.Skip("'ALLOYDB_AI_NL_PROJECT' not set")
	case AlloyDBAINLRegion:
		t.Skip("'ALLOYDB_AI_NL_REGION' not set")
	case AlloyDBAINLCluster:
		t.Skip("'ALLOYDB_AI_NL_CLUSTER' not set")
	case AlloyDBAINLInstance:
		t.Skip("'ALLOYDB_AI_NL_INSTANCE' not set")
	case AlloyDBAINLDatabase:
		t.Skip("'ALLOYDB_AI_NL_DATABASE' not set")
	case AlloyDBAINLUser:
		t.Skip("'ALLOYDB_AI_NL_USER' not set")
	case AlloyDBAINLPass:
		t.Skip("'ALLOYDB_AI_NL_PASS' not set")
	}
	return map[string]any{
		"type":     AlloyDBAINLSourceType,
		"project":  AlloyDBAINLProject,
		"cluster":  AlloyDBAINLCluster,
		"instance": AlloyDBAINLInstance,
		"region":   AlloyDBAINLRegion,
		"database": AlloyDBAINLDatabase,
		"user":     AlloyDBAINLUser,
		"password": AlloyDBAINLPass,
	}
}

func TestAlloyDBAINLToolEndpoints(t *testing.T) {
	sourceConfig := getAlloyDBAINLVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	// Write config into a file and pass it to command
	toolsFile := getAINLToolsConfig(sourceConfig)

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	runAINLToolInvokeTest(t)
	runAINLMCPToolCallMethod(t)
}

func runAINLToolInvokeTest(t *testing.T) {
	// Get ID token
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	sessionId := tests.RunInitialize(t, "2024-11-05")

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		toolName      string
		requestHeader map[string]string
		arguments     map[string]any
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-simple-tool",
			toolName:      "my-simple-tool",
			requestHeader: map[string]string{},
			arguments:     map[string]any{"question": "return the number 1"},
			want:          "[{\"execute_nl_query\":{\"?column?\":1}}]",
			isErr:         false,
		},
		{
			name:          "Invoke my-tool without parameters",
			toolName:      "my-tool",
			requestHeader: map[string]string{},
			arguments:     map[string]any{},
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-tool with auth token",
			toolName:      "my-auth-tool",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			arguments:     map[string]any{"question": "can you show me the name of this user?"},
			want:          "[{\"execute_nl_query\":{\"name\":\"Alice\"}}]",
			isErr:         false,
		},
		{
			name:          "Invoke my-auth-tool with invalid auth token",
			toolName:      "my-auth-tool",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			arguments:     map[string]any{"question": "return the number 1"},
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-tool without auth token",
			toolName:      "my-auth-tool",
			requestHeader: map[string]string{},
			arguments:     map[string]any{"question": "return the number 1"},
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-required-tool with auth token",
			toolName:      "my-auth-required-tool",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			arguments:     map[string]any{"question": "return the number 1"},
			isErr:         false,
			want:          "[{\"execute_nl_query\":{\"?column?\":1}}]",
		},
		{
			name:          "Invoke my-auth-required-tool with invalid auth token",
			toolName:      "my-auth-required-tool",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			arguments:     map[string]any{"question": "return the number 1"},
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-required-tool without auth token",
			toolName:      "my-auth-required-tool",
			requestHeader: map[string]string{},
			arguments:     map[string]any{"question": "return the number 1"},
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			mcpReq := map[string]any{
				"jsonrpc": "2.0",
				"id":      tc.toolName,
				"method":  "tools/call",
				"params": map[string]any{
					"name":      tc.toolName,
					"arguments": tc.arguments,
				},
			}
			reqBytes, _ := json.Marshal(mcpReq)

			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:5000/mcp", bytes.NewBuffer(reqBytes))
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			if sessionId != "" {
				req.Header.Add("Mcp-Session-Id", sessionId)
			}
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			bodyBytes, _ := io.ReadAll(resp.Body)

			var mcpResp struct {
				Error *struct {
					Message string `json:"message"`
				} `json:"error"`
				Result *struct {
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
					IsError bool `json:"isError"`
				} `json:"result"`
			}
			if err := json.Unmarshal(bodyBytes, &mcpResp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if mcpResp.Error != nil {
				if !tc.isErr {
					t.Fatalf("MCP returned an unexpected error: %s", mcpResp.Error.Message)
				}
				return
			}

			if tc.isErr {
				if mcpResp.Result != nil && mcpResp.Result.IsError {
					return
				}
				t.Fatalf("expected error but got valid response %s", string(bodyBytes))
			}

			if mcpResp.Result == nil || len(mcpResp.Result.Content) == 0 {
				t.Fatalf("unable to find result text in response body")
			}

			got := mcpResp.Result.Content[0].Text

			if !strings.Contains(got, tc.want) {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}

}

func getAINLToolsConfig(sourceConfig map[string]any) map[string]any {
	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"authServices": map[string]any{
			"my-google-auth": map[string]any{
				"type":     "google",
				"clientId": tests.ClientId,
			},
		},
		"tools": map[string]any{
			"my-simple-tool": map[string]any{
				"type":        AlloyDBAINLToolType,
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
				"nlConfig":    "my_nl_config",
			},
			"my-auth-tool": map[string]any{
				"type":        AlloyDBAINLToolType,
				"source":      "my-instance",
				"description": "Tool to test authenticated parameters.",
				"nlConfig":    "my_nl_config",
				"nlConfigParameters": []map[string]any{
					{
						"name":        "email",
						"type":        "string",
						"description": "user email",
						"authServices": []map[string]string{
							{
								"name":  "my-google-auth",
								"field": "email",
							},
						},
					},
				},
			},
			"my-auth-required-tool": map[string]any{
				"type":        AlloyDBAINLToolType,
				"source":      "my-instance",
				"description": "Tool to test auth required invocation.",
				"nlConfig":    "my_nl_config",
				"authRequired": []string{
					"my-google-auth",
				},
			},
		},
	}

	return toolsFile
}

func runAINLMCPToolCallMethod(t *testing.T) {
	sessionId := tests.RunInitialize(t, "2024-11-05")
	header := map[string]string{}
	if sessionId != "" {
		header["Mcp-Session-Id"] = sessionId
	}

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		api           string
		requestBody   jsonrpc.JSONRPCRequest
		requestHeader map[string]string
		want          string
	}{
		{
			name:          "MCP Invoke my-simple-tool",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "my-simple-tool",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name": "my-simple-tool",
					"arguments": map[string]any{
						"question": "return the number 1",
					},
				},
			},
			want: `{"jsonrpc":"2.0","id":"my-simple-tool","result":{"content":[{"type":"text","text":"{\"execute_nl_query\":{\"?column?\":1}}"}]}}`,
		},
		{
			name:          "MCP Invoke invalid tool",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invalid-tool",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "foo",
					"arguments": map[string]any{},
				},
			},
			want: `{"jsonrpc":"2.0","id":"invalid-tool","error":{"code":-32602,"message":"invalid tool name: tool with name \"foo\" does not exist"}}`,
		},
		{
			name:          "MCP Invoke my-auth-tool without parameters",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke-without-parameter",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "my-auth-tool",
					"arguments": map[string]any{},
				},
			},
			want: `{"jsonrpc":"2.0","id":"invoke-without-parameter","error":{"code":-32602,"message":"provided parameters were invalid: parameter question is required"}}`,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			reqMarshal, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("unexpected error during marshaling of request body")
			}
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, bytes.NewBuffer(reqMarshal))
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range header {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("unable to read request body: %s", err)
			}
			defer resp.Body.Close()
			got := string(bytes.TrimSpace(respBody))

			// Remove `\` and `"` for string comparison
			got = strings.ReplaceAll(got, "\\", "")
			want := strings.ReplaceAll(tc.want, "\\", "")
			got = strings.ReplaceAll(got, "\"", "")
			want = strings.ReplaceAll(want, "\"", "")

			if !strings.Contains(got, want) {
				t.Fatalf("Expected substring not found:\ngot:  %q\nwant: %q (to be contained within got)", got, want)
			}
		})
	}
}
