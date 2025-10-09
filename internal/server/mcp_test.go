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

package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
	"github.com/googleapis/genai-toolbox/internal/telemetry"
)

const jsonrpcVersion = "2.0"
const protocolVersion20241105 = "2024-11-05"
const protocolVersion20250326 = "2025-03-26"
const protocolVersion20250618 = "2025-06-18"
const serverName = "Toolbox"

var basicInputSchema = map[string]any{
	"type":       "object",
	"properties": map[string]any{},
	"required":   []any{},
}

var tool2InputSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"param1": map[string]any{"type": "integer", "description": "This is the first parameter."},
		"param2": map[string]any{"type": "integer", "description": "This is the second parameter."},
	},
	"required": []any{"param1", "param2"},
}

var tool3InputSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"my_array": map[string]any{
			"type":        "array",
			"description": "this param is an array of strings",
			"items":       map[string]any{"type": "string", "description": "string item"},
		},
	},
	"required": []any{"my_array"},
}

var prompt2Args = []any{
	map[string]any{
		"name":        "arg1",
		"description": "This is the first argument.",
		"required":    true,
	},
}

func TestMcpEndpointWithoutInitialized(t *testing.T) {
	mockTools := []MockTool{tool1, tool2, tool3, tool4, tool5}
	mockPrompts := []MockPrompt{prompt1, prompt2}
	toolsMap, toolsets, promptsMap, promptsets := setUpResources(t, mockTools, mockPrompts)
	r, shutdown := setUpServer(t, "mcp", toolsMap, toolsets, promptsMap, promptsets)
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	testCases := []struct {
		name  string
		url   string
		isErr bool
		body  jsonrpc.JSONRPCRequest
		want  map[string]any
	}{
		{
			name: "ping",
			url:  "/",
			body: jsonrpc.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Id:      "ping-test-123",
				Request: jsonrpc.Request{
					Method: "ping",
				},
			},
			isErr: false,
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      "ping-test-123",
				"result":  map[string]any{},
			},
		},
		{
			name: "tools/list",
			url:  "/",
			body: jsonrpc.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Id:      "tools-list",
				Request: jsonrpc.Request{
					Method: "tools/list",
				},
			},
			isErr: false,
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      "tools-list",
				"result": map[string]any{
					"tools": []any{
						map[string]any{
							"name":        "no_params",
							"inputSchema": basicInputSchema,
						},
						map[string]any{
							"name":        "some_params",
							"inputSchema": tool2InputSchema,
						},
						map[string]any{
							"name":        "array_param",
							"description": "some description",
							"inputSchema": tool3InputSchema,
						},
						map[string]any{
							"name":        "unauthorized_tool",
							"inputSchema": basicInputSchema,
						},
						map[string]any{
							"name":        "require_client_auth_tool",
							"inputSchema": basicInputSchema,
						},
					},
				},
			},
		},
		{
			name:  "missing method",
			url:   "/",
			isErr: true,
			body: jsonrpc.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Id:      "missing-method",
				Request: jsonrpc.Request{},
			},
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      "missing-method",
				"error": map[string]any{
					"code":    -32601.0,
					"message": "method not found",
				},
			},
		},
		{
			name:  "invalid jsonrpc version",
			url:   "/",
			isErr: true,
			body: jsonrpc.JSONRPCRequest{
				Jsonrpc: "1.0",
				Id:      "invalid-jsonrpc-version",
				Request: jsonrpc.Request{
					Method: "foo",
				},
			},
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      "invalid-jsonrpc-version",
				"error": map[string]any{
					"code":    -32600.0,
					"message": "invalid json-rpc version",
				},
			},
		},
		{
			name: "call tool1 unauthorized tool",
			url:  "/",
			body: jsonrpc.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Id:      "tools-call-tool1",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name": "no_params",
				},
			},
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      "tools-call-tool1",
				"result": map[string]any{
					"content": []any{
						map[string]any{
							"type": "text",
							"text": `"no_params"`,
						},
					},
				},
			},
		},
		{
			name: "call tool4 unauthorized tool",
			url:  "/",
			body: jsonrpc.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Id:      "tools-call-tool4",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name": "unauthorized_tool",
				},
			},
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      "tools-call-tool4",
				"error": map[string]any{
					"code":    -32600.0,
					"message": "unauthorized Tool call: Please make sure your specify correct auth headers: unauthorized",
				},
			},
		},
		{
			name: "call tool5 unauthorized tool",
			url:  "/",
			body: jsonrpc.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Id:      "tools-call-tool5",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name": "require_client_auth_tool",
				},
			},
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      "tools-call-tool5",
				"error": map[string]any{
					"code":    -32600.0,
					"message": "missing access token in the 'Authorization' header",
				},
			},
		},
		{
			name: "prompts/list",
			url:  "/",
			body: jsonrpc.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Id:      "prompts-list-uninitialized",
				Request: jsonrpc.Request{
					Method: "prompts/list",
				},
			},
			isErr: false,
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      "prompts-list-uninitialized",
				"result": map[string]any{
					"prompts": []any{
						map[string]any{
							"name": "prompt1",
						},
						map[string]any{
							"name":      "prompt2",
							"arguments": prompt2Args,
						},
					},
				},
			},
		},
		{
			name:  "prompts/get non-existent prompt",
			url:   "/",
			isErr: true,
			body: jsonrpc.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Id:      "prompts-get-non-existent",
				Request: jsonrpc.Request{
					Method: "prompts/get",
				},
				Params: map[string]any{
					"name": "non_existent_prompt",
				},
			},
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      "prompts-get-non-existent",
				"error": map[string]any{
					"code":    -32602.0,
					"message": `prompt with name "non_existent_prompt" does not exist`,
				},
			},
		},
		{
			name:  "prompts/get with invalid arguments",
			url:   "/",
			isErr: true,
			body: jsonrpc.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Id:      "prompts-get-invalid-args",
				Request: jsonrpc.Request{
					Method: "prompts/get",
				},
				Params: map[string]any{
					"name": "prompt2",
					"arguments": map[string]any{
						"arg1": 42, // prompt2 expects a string, we send a number
					},
				},
			},
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      "prompts-get-invalid-args",
				"error": map[string]any{
					"code":    -32602.0,
					"message": `invalid arguments for prompt "prompt2": unable to parse value for "arg1": %!q(float64=42) not type "string"`,
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqMarshal, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatalf("unexpected error during marshaling of body")
			}

			resp, body, err := runRequest(ts, http.MethodPost, tc.url, bytes.NewBuffer(reqMarshal), nil)
			if err != nil {
				t.Fatalf("unexpected error during request: %s", err)
			}

			// Notifications don't expect a response.
			if tc.want != nil {
				if contentType := resp.Header.Get("Content-type"); contentType != "application/json" {
					t.Fatalf("unexpected content-type header: want %s, got %s", "application/json", contentType)
				}

				var got map[string]any
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("unexpected error unmarshalling body: %s", err)
				}
				if !reflect.DeepEqual(got, tc.want) {
					t.Fatalf("unexpected response: got %+v, want %+v", got, tc.want)
				}
			}
		})
	}
}

func runInitializeLifecycle(t *testing.T, ts *httptest.Server, protocolVersion string, initializeWant map[string]any, idHeader bool) string {
	initializeRequestBody := map[string]any{
		"jsonrpc": jsonrpcVersion,
		"id":      "mcp-initialize",
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": protocolVersion,
		},
	}
	reqMarshal, err := json.Marshal(initializeRequestBody)
	if err != nil {
		t.Fatalf("unexpected error during marshaling of body")
	}

	resp, body, err := runRequest(ts, http.MethodPost, "/", bytes.NewBuffer(reqMarshal), nil)
	if err != nil {
		t.Fatalf("unexpected error during request: %s", err)
	}

	if contentType := resp.Header.Get("Content-type"); contentType != "application/json" {
		t.Fatalf("unexpected content-type header: want %s, got %s", "application/json", contentType)
	}

	sessionId := resp.Header.Get("Mcp-Session-Id")
	if idHeader && sessionId == "" {
		t.Fatalf("Mcp-Session-Id header is expected")
	}

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unexpected error unmarshalling body: %s", err)
	}
	if !reflect.DeepEqual(got, initializeWant) {
		t.Fatalf("unexpected response: got %+v, want %+v", got, initializeWant)
	}

	header := map[string]string{}
	if sessionId != "" {
		header["Mcp-Session-Id"] = sessionId
	}

	initializeNotificationBody := map[string]any{
		"jsonrpc": jsonrpcVersion,
		"method":  "notifications/initialized",
	}
	notiMarshal, err := json.Marshal(initializeNotificationBody)
	if err != nil {
		t.Fatalf("unexpected error during marshaling of notifications body")
	}

	_, _, err = runRequest(ts, http.MethodPost, "/", bytes.NewBuffer(notiMarshal), header)
	if err != nil {
		t.Fatalf("unexpected error during request: %s", err)
	}
	return sessionId
}

func TestMcpEndpoint(t *testing.T) {
	mockTools := []MockTool{tool1, tool2, tool3, tool4, tool5}
	mockPrompts := []MockPrompt{prompt1, prompt2}
	toolsMap, toolsets, promptsMap, promptsets := setUpResources(t, mockTools, mockPrompts)
	r, shutdown := setUpServer(t, "mcp", toolsMap, toolsets, promptsMap, promptsets)
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	versTestCases := []struct {
		name     string
		protocol string
		idHeader bool
		initWant map[string]any
	}{
		{
			name:     "version 2024-11-05",
			protocol: protocolVersion20241105,
			idHeader: false,
			initWant: map[string]any{
				"jsonrpc": "2.0",
				"id":      "mcp-initialize",
				"result": map[string]any{
					"protocolVersion": "2024-11-05",
					"capabilities": map[string]any{
						"tools":   map[string]any{"listChanged": false},
						"prompts": map[string]any{"listChanged": false},
					},
					"serverInfo": map[string]any{"name": serverName, "version": fakeVersionString},
				},
			},
		},
		{
			name:     "version 2025-03-26",
			protocol: protocolVersion20250326,
			idHeader: true,
			initWant: map[string]any{
				"jsonrpc": "2.0",
				"id":      "mcp-initialize",
				"result": map[string]any{
					"protocolVersion": "2025-03-26",
					"capabilities": map[string]any{
						"tools":   map[string]any{"listChanged": false},
						"prompts": map[string]any{"listChanged": false},
					},
					"serverInfo": map[string]any{"name": serverName, "version": fakeVersionString},
				},
			},
		},
		{
			name:     "version 2025-06-18",
			protocol: protocolVersion20250618,
			idHeader: false,
			initWant: map[string]any{
				"jsonrpc": "2.0",
				"id":      "mcp-initialize",
				"result": map[string]any{
					"protocolVersion": "2025-06-18",
					"capabilities": map[string]any{
						"tools":   map[string]any{"listChanged": false},
						"prompts": map[string]any{"listChanged": false},
					},
					"serverInfo": map[string]any{"name": serverName, "version": fakeVersionString},
				},
			},
		},
	}
	for _, vtc := range versTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			sessionId := runInitializeLifecycle(t, ts, vtc.protocol, vtc.initWant, vtc.idHeader)

			header := map[string]string{}
			if sessionId != "" {
				header["Mcp-Session-Id"] = sessionId
			}

			if vtc.protocol == protocolVersion20250618 {
				header["MCP-Protocol-Version"] = vtc.protocol
			}

			testCases := []struct {
				name           string
				url            string
				isErr          bool
				body           any
				wantStatusCode int
				want           map[string]any
			}{
				{
					name: "basic notification",
					url:  "/",
					body: jsonrpc.JSONRPCRequest{
						Jsonrpc: jsonrpcVersion,
						Request: jsonrpc.Request{
							Method: "notification",
						},
					},
					wantStatusCode: http.StatusAccepted,
				},
				{
					name: "ping",
					url:  "/",
					body: jsonrpc.JSONRPCRequest{
						Jsonrpc: jsonrpcVersion,
						Id:      "ping-test-123",
						Request: jsonrpc.Request{
							Method: "ping",
						},
					},
					wantStatusCode: http.StatusOK,
					want: map[string]any{
						"jsonrpc": "2.0",
						"id":      "ping-test-123",
						"result":  map[string]any{},
					},
				},
				{
					name: "tools/list",
					url:  "/",
					body: jsonrpc.JSONRPCRequest{
						Jsonrpc: jsonrpcVersion,
						Id:      "tools-list",
						Request: jsonrpc.Request{
							Method: "tools/list",
						},
					},
					wantStatusCode: http.StatusOK,
					want: map[string]any{
						"jsonrpc": "2.0",
						"id":      "tools-list",
						"result": map[string]any{
							"tools": []any{
								map[string]any{
									"name":        "no_params",
									"inputSchema": basicInputSchema,
								},
								map[string]any{
									"name":        "some_params",
									"inputSchema": tool2InputSchema,
								},
								map[string]any{
									"name":        "array_param",
									"description": "some description",
									"inputSchema": tool3InputSchema,
								},
								map[string]any{
									"name":        "unauthorized_tool",
									"inputSchema": basicInputSchema,
								},
								map[string]any{
									"name":        "require_client_auth_tool",
									"inputSchema": basicInputSchema,
								},
							},
						},
					},
				},
				{
					name: "prompts/list",
					url:  "/",
					body: jsonrpc.JSONRPCRequest{
						Jsonrpc: jsonrpcVersion,
						Id:      "prompts-list",
						Request: jsonrpc.Request{
							Method: "prompts/list",
						},
					},
					wantStatusCode: http.StatusOK,
					want: map[string]any{
						"jsonrpc": "2.0",
						"id":      "prompts-list",
						"result": map[string]any{
							"prompts": []any{
								map[string]any{
									"name": "prompt1",
								},
								map[string]any{
									"name":      "prompt2",
									"arguments": prompt2Args,
								},
							},
						},
					},
				},
				{
					name: "prompts/get",
					url:  "/",
					body: jsonrpc.JSONRPCRequest{
						Jsonrpc: jsonrpcVersion,
						Id:      "prompts-get-prompt2",
						Request: jsonrpc.Request{
							Method: "prompts/get",
						},
						Params: map[string]any{
							"name": "prompt2",
							"arguments": map[string]any{
								"arg1": "value1",
							},
						},
					},
					wantStatusCode: http.StatusOK,
					want: map[string]any{
						"jsonrpc": "2.0",
						"id":      "prompts-get-prompt2",
						"result": map[string]any{
							"messages": []any{
								map[string]any{
									"role": "user",
									"content": map[string]any{
										"type": "text",
										"text": "substituted prompt2",
									},
								},
							},
						},
					},
				},
				{
					name: "tools/list on tool1_only",
					url:  "/tool1_only",
					body: jsonrpc.JSONRPCRequest{
						Jsonrpc: jsonrpcVersion,
						Id:      "tools-list-tool1",
						Request: jsonrpc.Request{
							Method: "tools/list",
						},
					},
					wantStatusCode: http.StatusOK,
					want: map[string]any{
						"jsonrpc": "2.0",
						"id":      "tools-list-tool1",
						"result": map[string]any{
							"tools": []any{
								map[string]any{
									"name":        "no_params",
									"inputSchema": basicInputSchema,
								},
							},
						},
					},
				},
				{
					name:  "tools/list on invalid tool set",
					url:   "/foo",
					isErr: true,
					body: jsonrpc.JSONRPCRequest{
						Jsonrpc: jsonrpcVersion,
						Id:      "tools-list-invalid-toolset",
						Request: jsonrpc.Request{
							Method: "tools/list",
						},
					},
					wantStatusCode: http.StatusOK,
					want: map[string]any{
						"jsonrpc": "2.0",
						"id":      "tools-list-invalid-toolset",
						"error": map[string]any{
							"code":    -32600.0,
							"message": "toolset does not exist",
						},
					},
				},
				{
					name:  "missing method",
					url:   "/",
					isErr: true,
					body: jsonrpc.JSONRPCRequest{
						Jsonrpc: jsonrpcVersion,
						Id:      "missing-method",
						Request: jsonrpc.Request{},
					},
					wantStatusCode: http.StatusOK,
					want: map[string]any{
						"jsonrpc": "2.0",
						"id":      "missing-method",
						"error": map[string]any{
							"code":    -32601.0,
							"message": "method not found",
						},
					},
				},
				{
					name:  "invalid method",
					url:   "/",
					isErr: true,
					body: jsonrpc.JSONRPCRequest{
						Jsonrpc: jsonrpcVersion,
						Id:      "invalid-method",
						Request: jsonrpc.Request{
							Method: "foo",
						},
					},
					wantStatusCode: http.StatusOK,
					want: map[string]any{
						"jsonrpc": "2.0",
						"id":      "invalid-method",
						"error": map[string]any{
							"code":    -32601.0,
							"message": "invalid method foo",
						},
					},
				},
				{
					name:  "invalid jsonrpc version",
					url:   "/",
					isErr: true,
					body: jsonrpc.JSONRPCRequest{
						Jsonrpc: "1.0",
						Id:      "invalid-jsonrpc-version",
						Request: jsonrpc.Request{
							Method: "foo",
						},
					},
					wantStatusCode: http.StatusOK,
					want: map[string]any{
						"jsonrpc": "2.0",
						"id":      "invalid-jsonrpc-version",
						"error": map[string]any{
							"code":    -32600.0,
							"message": "invalid json-rpc version",
						},
					},
				},
				{
					name:  "batch requests",
					url:   "/",
					isErr: true,
					body: []any{
						jsonrpc.JSONRPCRequest{
							Jsonrpc: "1.0",
							Id:      "batch-requests1",
							Request: jsonrpc.Request{
								Method: "foo",
							},
						},
						jsonrpc.JSONRPCRequest{
							Jsonrpc: jsonrpcVersion,
							Id:      "batch-requests2",
							Request: jsonrpc.Request{
								Method: "tools/list",
							},
						},
					},
					wantStatusCode: http.StatusOK,
					want: map[string]any{
						"jsonrpc": "2.0",
						"error": map[string]any{
							"code":    -32600.0,
							"message": "not supporting batch requests",
						},
					},
				},
				{
					name: "call tool1 unauthorized tool",
					url:  "/",
					body: jsonrpc.JSONRPCRequest{
						Jsonrpc: jsonrpcVersion,
						Id:      "tools-call-tool1",
						Request: jsonrpc.Request{
							Method: "tools/call",
						},
						Params: map[string]any{
							"name": "no_params",
						},
					},
					wantStatusCode: http.StatusOK,
					want: map[string]any{
						"jsonrpc": "2.0",
						"id":      "tools-call-tool1",
						"result": map[string]any{
							"content": []any{
								map[string]any{
									"type": "text",
									"text": `"no_params"`,
								},
							},
						},
					},
				},
				{
					name: "call tool4 unauthorized tool",
					url:  "/",
					body: jsonrpc.JSONRPCRequest{
						Jsonrpc: jsonrpcVersion,
						Id:      "tools-call-tool4",
						Request: jsonrpc.Request{
							Method: "tools/call",
						},
						Params: map[string]any{
							"name": "unauthorized_tool",
						},
					},
					wantStatusCode: http.StatusUnauthorized,
					want: map[string]any{
						"jsonrpc": "2.0",
						"id":      "tools-call-tool4",
						"error": map[string]any{
							"code":    -32600.0,
							"message": "unauthorized Tool call: Please make sure your specify correct auth headers: unauthorized",
						},
					},
				},
				{
					name: "call tool5 unauthorized tool",
					url:  "/",
					body: jsonrpc.JSONRPCRequest{
						Jsonrpc: jsonrpcVersion,
						Id:      "tools-call-tool5",
						Request: jsonrpc.Request{
							Method: "tools/call",
						},
						Params: map[string]any{
							"name": "require_client_auth_tool",
						},
					},
					wantStatusCode: http.StatusUnauthorized,
					want: map[string]any{
						"jsonrpc": "2.0",
						"id":      "tools-call-tool5",
						"error": map[string]any{
							"code":    -32600.0,
							"message": "missing access token in the 'Authorization' header",
						},
					},
				},
			}
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					reqMarshal, err := json.Marshal(tc.body)
					if err != nil {
						t.Fatalf("unexpected error during marshaling of body")
					}

					if vtc.protocol != protocolVersion20241105 && len(header) == 0 {
						t.Fatalf("header is missing")
					}

					resp, body, err := runRequest(ts, http.MethodPost, tc.url, bytes.NewBuffer(reqMarshal), header)

					if err != nil {
						t.Fatalf("unexpected error during request: %s", err)
					}

					if resp.StatusCode != tc.wantStatusCode {
						t.Errorf("StatusCode mismatch: got %d, want %d", resp.StatusCode, tc.wantStatusCode)
					}

					// Notifications don't expect a response.
					if tc.want != nil {
						if contentType := resp.Header.Get("Content-type"); contentType != "application/json" {
							t.Fatalf("unexpected content-type header: want %s, got %s", "application/json", contentType)
						}

						var got map[string]any
						if err := json.Unmarshal(body, &got); err != nil {
							t.Fatalf("unexpected error unmarshalling body: %s", err)
						}
						// for decode failure, a random uuid is generated in server
						if tc.want["id"] == nil {
							tc.want["id"] = got["id"]
						}
						if !reflect.DeepEqual(got, tc.want) {
							t.Fatalf("unexpected response: got %+v, want %+v", got, tc.want)
						}
					}
				})
			}
		})
	}
}

func TestInvalidProtocolVersionHeader(t *testing.T) {
	r, shutdown := setUpServer(t, "mcp", nil, nil, nil, nil)
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	header := map[string]string{}
	header["MCP-Protocol-Version"] = "foo"

	resp, body, err := runRequest(ts, http.MethodPost, "/", nil, header)
	if resp.Status != "400 Bad Request" {
		t.Fatalf("unexpected status: %s", resp.Status)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unexpected error unmarshalling body: %s", err)
	}
	want := "invalid protocol version: foo"
	if got["error"] != want {
		t.Fatalf("unexpected error message: got %s, want %s", got["error"], want)
	}
	if err != nil {
		t.Fatalf("unexpected error during request: %s", err)
	}
}

func TestDeleteEndpoint(t *testing.T) {
	r, shutdown := setUpServer(t, "mcp", nil, nil, nil, nil)
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	resp, _, err := runRequest(ts, http.MethodDelete, "/", nil, nil)
	if resp.Status != "200 OK" {
		t.Fatalf("unexpected status: %s", resp.Status)
	}
	if err != nil {
		t.Fatalf("unexpected error during request: %s", err)
	}
}

func TestGetEndpoint(t *testing.T) {
	r, shutdown := setUpServer(t, "mcp", nil, nil, nil, nil)
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	resp, body, err := runRequest(ts, http.MethodGet, "/", nil, nil)
	if resp.Status != "405 Method Not Allowed" {
		t.Fatalf("unexpected status: %s", resp.Status)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unexpected error unmarshalling body: %s", err)
	}
	want := "toolbox does not support streaming in streamable HTTP transport"
	if got["error"] != want {
		t.Fatalf("unexpected error message: %s", got["error"])
	}
	if err != nil {
		t.Fatalf("unexpected error during request: %s", err)
	}
}

func TestSseEndpoint(t *testing.T) {
	r, shutdown := setUpServer(t, "mcp", nil, nil, nil, nil)
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()
	if !strings.Contains(ts.URL, "http://127.0.0.1") {
		t.Fatalf("unexpected url, got %s", ts.URL)
	}
	tsPort := strings.TrimPrefix(ts.URL, "http://127.0.0.1:")
	tls := runServer(r, true)
	defer tls.Close()
	if !strings.Contains(tls.URL, "https://127.0.0.1") {
		t.Fatalf("unexpected url, got %s", tls.URL)
	}
	tlsPort := strings.TrimPrefix(tls.URL, "https://127.0.0.1:")

	contentType := "text/event-stream"
	cacheControl := "no-cache"
	connection := "keep-alive"
	accessControlAllowOrigin := "*"

	testCases := []struct {
		name   string
		server *httptest.Server
		path   string
		proto  string
		event  string
	}{
		{
			name:   "basic",
			server: ts,
			path:   "/sse",
			event:  fmt.Sprintf("event: endpoint\ndata: %s/mcp?sessionId=", ts.URL),
		},
		{
			name:   "toolset1",
			server: ts,
			path:   "/tool1_only/sse",
			event:  fmt.Sprintf("event: endpoint\ndata: http://127.0.0.1:%s/mcp/tool1_only?sessionId=", tsPort),
		},
		{
			name:   "promptset1",
			server: ts,
			path:   "/prompt1_only/sse",
			event:  fmt.Sprintf("event: endpoint\ndata: http://127.0.0.1:%s/mcp/prompt1_only?sessionId=", tsPort),
		},
		{
			name:   "basic with http proto",
			server: ts,
			path:   "/sse",
			proto:  "http",
			event:  fmt.Sprintf("event: endpoint\ndata: http://127.0.0.1:%s/mcp?sessionId=", tsPort),
		},
		{
			name:   "basic tls with https proto",
			server: ts,
			path:   "/sse",
			proto:  "https",
			event:  fmt.Sprintf("event: endpoint\ndata: https://127.0.0.1:%s/mcp?sessionId=", tsPort),
		},
		{
			name:   "basic tls",
			server: tls,
			path:   "/sse",
			event:  fmt.Sprintf("event: endpoint\ndata: https://127.0.0.1:%s/mcp?sessionId=", tlsPort),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := runSseRequest(tc.server, tc.path, tc.proto)
			if err != nil {
				t.Fatalf("unable to run sse request: %s", err)
			}
			defer resp.Body.Close()

			if gotContentType := resp.Header.Get("Content-type"); gotContentType != contentType {
				t.Fatalf("unexpected content-type header: want %s, got %s", contentType, gotContentType)
			}
			if gotCacheControl := resp.Header.Get("Cache-Control"); gotCacheControl != cacheControl {
				t.Fatalf("unexpected cache-control header: want %s, got %s", cacheControl, gotCacheControl)
			}
			if gotConnection := resp.Header.Get("Connection"); gotConnection != connection {
				t.Fatalf("unexpected content-type header: want %s, got %s", connection, gotConnection)
			}
			if gotAccessControlAllowOrigin := resp.Header.Get("Access-Control-Allow-Origin"); gotAccessControlAllowOrigin != accessControlAllowOrigin {
				t.Fatalf("unexpected cache-control header: want %s, got %s", accessControlAllowOrigin, gotAccessControlAllowOrigin)
			}

			buffer := make([]byte, 1024)
			n, err := resp.Body.Read(buffer)
			if err != nil {
				t.Fatalf("unable to read response: %s", err)
			}
			endpointEvent := string(buffer[:n])
			if !strings.Contains(endpointEvent, tc.event) {
				t.Fatalf("unexpected event: got %s, want to contain %s", endpointEvent, tc.event)
			}
		})
	}
}

func runSseRequest(ts *httptest.Server, path string, proto string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}
	if proto != "" {
		req.Header.Set("X-Forwarded-Proto", proto)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to send request: %w", err)
	}
	return resp, nil
}

func TestStdioSession(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockTools := []MockTool{tool1, tool2, tool3}
	mockPrompts := []MockPrompt{prompt1, prompt2}
	toolsMap, toolsets, promptsMap, promptsets := setUpResources(t, mockTools, mockPrompts)

	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("error with Pipe: %s", err)
	}

	testLogger, err := log.NewStdLogger(pw, os.Stderr, "warn")
	if err != nil {
		t.Fatalf("unable to initialize logger: %s", err)
	}

	otelShutdown, err := telemetry.SetupOTel(ctx, fakeVersionString, "", false, "toolbox")
	if err != nil {
		t.Fatalf("unable to setup otel: %s", err)
	}
	defer func() {
		err := otelShutdown(ctx)
		if err != nil {
			t.Fatalf("error shutting down OpenTelemetry: %s", err)
		}
	}()

	instrumentation, err := telemetry.CreateTelemetryInstrumentation(fakeVersionString)
	if err != nil {
		t.Fatalf("unable to create custom metrics: %s", err)
	}

	sseManager := newSseManager(ctx)

	resourceManager := NewResourceManager(nil, nil, toolsMap, toolsets, promptsMap, promptsets)

	server := &Server{
		version:         fakeVersionString,
		logger:          testLogger,
		instrumentation: instrumentation,
		sseManager:      sseManager,
		ResourceMgr:     resourceManager,
	}

	in := bufio.NewReader(pr)
	stdioSession := NewStdioSession(server, in, pw)

	// test stdioSession.readLine()
	input := "test readLine function\n"
	_, err = fmt.Fprintf(pw, "%s", input)
	if err != nil {
		t.Fatalf("error writing into pipe w: %s", err)
	}

	line, err := stdioSession.readLine(ctx)
	if err != nil {
		t.Fatalf("error with stdioSession.readLine: %s", err)
	}
	if line != input {
		t.Fatalf("unexpected line: got %s, want %s", line, input)
	}

	// test stdioSession.write()
	write := "test write function"
	err = stdioSession.write(ctx, write)
	if err != nil {
		t.Fatalf("error with stdioSession.write: %s", err)
	}

	read, err := in.ReadString('\n')
	if err != nil {
		t.Fatalf("error reading: %s", err)
	}
	want := fmt.Sprintf(`"%s"`, write) + "\n"
	if read != want {
		t.Fatalf("unexpected read: got %s, want %s", read, want)
	}
}
