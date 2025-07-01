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
	"sync"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/server/mcp"
	"github.com/googleapis/genai-toolbox/internal/telemetry"
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
	r, shutdown := setUpServer(t, "mcp", toolsMap, toolsets)
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	testCases := []struct {
		name  string
		url   string
		isErr bool
		body  mcp.JSONRPCRequest
		want  map[string]any
	}{
		{
			name: "initialize",
			url:  "/",
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
			url:  "/",
			body: mcp.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Request: mcp.Request{
					Method: "notification",
				},
			},
		},
		{
			name: "tools/list",
			url:  "/",
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
			name: "tools/list on tool1_only",
			url:  "/tool1_only",
			body: mcp.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Id:      "tools-list-tool1",
				Request: mcp.Request{
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
			body: mcp.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Id:      "tools-list-invalid-toolset",
				Request: mcp.Request{
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
			url:   "/",
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
			url:   "/",
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

			resp, body, err := runRequest(ts, http.MethodPost, tc.url, bytes.NewBuffer(reqMarshal))
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

	instrumentation, err := telemetry.CreateTelemetryInstrumentation(fakeVersionString)
	if err != nil {
		t.Fatalf("unable to create custom metrics: %s", err)
	}

	sseManager := &sseManager{
		mu:          sync.RWMutex{},
		sseSessions: make(map[string]*sseSession),
	}

	resourceManager := NewResourceManager(nil, nil, toolsMap, toolsets)

	server := &Server{
		version:         fakeVersionString,
		logger:          testLogger,
		instrumentation: instrumentation,
		sseManager:      sseManager,
		resourceMgr:     resourceManager,
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
