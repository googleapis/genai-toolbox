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
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/server/mcp"
)

const jsonrpcVersion = "2.0"
const protocolVersion = "2024-11-05"
const serverName = "Toolbox"

var tool1InputSchema = map[string]any{
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

func TestMcpEndpoint(t *testing.T) {
	mockTools := []MockTool{tool1, tool2, tool3}
	toolsMap, toolsets := setUpResources(t, mockTools)
	ts, shutdown := setUpServer(t, "mcp", toolsMap, toolsets)
	defer shutdown()

	testCases := []struct {
		name  string
		isErr bool
		body  mcp.JSONRPCRequest
		want  map[string]any
	}{
		{
			name: "initialize",
			body: mcp.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Id:      "mcp-initialize",
				Request: mcp.Request{
					Method: "initialize",
				},
			},
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      "mcp-initialize",
				"result": map[string]any{
					"protocolVersion": protocolVersion,
					"capabilities": map[string]any{
						"tools": map[string]any{"listChanged": false},
					},
					"serverInfo": map[string]any{"name": serverName, "version": fakeVersionString},
				},
			},
		},
		{
			name: "basic notification",
			body: mcp.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Request: mcp.Request{
					Method: "notification",
				},
			},
		},
		{
			name: "tools/list",
			body: mcp.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Id:      "tools-list",
				Request: mcp.Request{
					Method: "tools/list",
				},
			},
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      "tools-list",
				"result": map[string]any{
					"tools": []any{
						map[string]any{
							"name":        "no_params",
							"inputSchema": tool1InputSchema,
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
					},
				},
			},
		},
		{
			name:  "missing method",
			isErr: true,
			body: mcp.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Id:      "missing-method",
				Request: mcp.Request{},
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
			name:  "invalid method",
			isErr: true,
			body: mcp.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Id:      "invalid-method",
				Request: mcp.Request{
					Method: "foo",
				},
			},
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
			isErr: true,
			body: mcp.JSONRPCRequest{
				Jsonrpc: "1.0",
				Id:      "invalid-jsonrpc-version",
				Request: mcp.Request{
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
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqMarshal, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatalf("unexpected error during marshaling of body")
			}

			resp, body, err := runRequest(ts, http.MethodPost, "/", bytes.NewBuffer(reqMarshal))
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

func TestSseEndpoint(t *testing.T) {
	ts, shutdown := setUpServer(t, "mcp", nil, nil)
	defer shutdown()

	contentType := "text/event-stream"
	cacheControl := "no-cache"
	connection := "keep-alive"
	accessControlAllowOrigin := "*"
	wantEvent := "event: endpoint"

	t.Run("test sse endpoint", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/sse")
		if err != nil {
			t.Fatalf("unexpected error during request: %s", err)
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
		if !strings.Contains(endpointEvent, wantEvent) {
			t.Fatalf("unexpected event: got %s", endpointEvent)
		}
	})
}
