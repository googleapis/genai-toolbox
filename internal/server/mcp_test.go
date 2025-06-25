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
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const jsonrpcVersion = "2.0"
const protocolVersion20241105 = "2024-11-05"
const protocolVersion20250326 = "2025-03-26"
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

func TestMcpEndpointWithoutInitialized(t *testing.T) {
	mockTools := []MockTool{tool1, tool2, tool3}
	toolsMap, toolsets := setUpResources(t, mockTools)
	r, shutdown := setUpServer(t, "mcp", toolsMap, toolsets)
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
	mockTools := []MockTool{tool1, tool2, tool3}
	toolsMap, toolsets := setUpResources(t, mockTools)
	r, shutdown := setUpServer(t, "mcp", toolsMap, toolsets)
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
			name:     "verson 2024-11-05",
			protocol: protocolVersion20241105,
			idHeader: false,
			initWant: map[string]any{
				"jsonrpc": "2.0",
				"id":      "mcp-initialize",
				"result": map[string]any{
					"protocolVersion": "2024-11-05",
					"capabilities": map[string]any{
						"tools": map[string]any{"listChanged": false},
					},
					"serverInfo": map[string]any{"name": serverName, "version": fakeVersionString},
				},
			},
		},
		{
			name:     "verson 2025-03-26",
			protocol: protocolVersion20250326,
			idHeader: true,
			initWant: map[string]any{
				"jsonrpc": "2.0",
				"id":      "mcp-initialize",
				"result": map[string]any{
					"protocolVersion": "2025-03-26",
					"capabilities": map[string]any{
						"tools": map[string]any{"listChanged": false},
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

			testCases := []struct {
				name  string
				url   string
				isErr bool
				body  jsonrpc.JSONRPCRequest
				want  map[string]any
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
					name: "tools/list on tool1_only",
					url:  "/tool1_only",
					body: jsonrpc.JSONRPCRequest{
						Jsonrpc: jsonrpcVersion,
						Id:      "tools-list-tool1",
						Request: jsonrpc.Request{
							Method: "tools/list",
						},
					},
					want: map[string]any{
						"jsonrpc": "2.0",
						"id":      "tools-list-tool1",
						"result": map[string]any{
							"tools": []any{
								map[string]any{
									"name":        "no_params",
									"inputSchema": tool1InputSchema,
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

					if vtc.protocol == protocolVersion20250326 && len(header) == 0 {
						t.Fatalf("header is missing")
					}

					resp, body, err := runRequest(ts, http.MethodPost, tc.url, bytes.NewBuffer(reqMarshal), header)
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
		})
	}
}

func TestDeleteEndpoint(t *testing.T) {
	toolsMap, toolsets := map[string]tools.Tool{}, map[string]tools.Toolset{}
	r, shutdown := setUpServer(t, "mcp", toolsMap, toolsets)
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
	toolsMap, toolsets := map[string]tools.Tool{}, map[string]tools.Toolset{}
	r, shutdown := setUpServer(t, "mcp", toolsMap, toolsets)
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
	r, shutdown := setUpServer(t, "mcp", nil, nil)
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
	toolsMap, toolsets := setUpResources(t, mockTools)

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

	instrumentation, err := CreateTelemetryInstrumentation(fakeVersionString)
	if err != nil {
		t.Fatalf("unable to create custom metrics: %s", err)
	}

	sseManager := newSseManager(ctx)

	server := &Server{version: fakeVersionString, logger: testLogger, instrumentation: instrumentation, sseManager: sseManager, tools: toolsMap, toolsets: toolsets}

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
