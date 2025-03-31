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
	"testing"

	"github.com/googleapis/genai-toolbox/internal/server/mcp"
)

const jsonrpcVersion = "2.0"

func TestMcpEndpoint(t *testing.T) {
	toolsMap, toolsets := setUpResources(t)
	ts, shutdown := setUpServer(t, "mcp", toolsMap, toolsets)
	defer shutdown()

	testCases := []struct {
		name  string
		isErr bool
		body  mcp.JSONRPCRequest
		want  map[string]any
	}{
		{
			name:  "basic mcp",
			isErr: false,
			body: mcp.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Id:      "basic-mcp",
				Request: mcp.Request{
					Method: "foo",
				},
			},
			want: map[string]any{
				"jsonrpc": "2.0",
				"id":      "basic-mcp",
				"result":  map[string]any{},
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
		})
	}
}
