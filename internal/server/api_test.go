// Copyright 2024 Google LLC
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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/server/mcp"
	"github.com/googleapis/genai-toolbox/internal/telemetry"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

var _ tools.Tool = &MockTool{}

const fakeVersionString = "0.0.0"
const jsonrpcVersion = "2.0"
const protocolVersion = "2024-11-05"
const serverName = "Toolbox"

type MockTool struct {
	Name        string
	Description string
	Params      []tools.Parameter
	manifest    tools.Manifest
}

var tool1 = MockTool{
	Name:   "no_params",
	Params: []tools.Parameter{},
}

var tool2 = MockTool{
	Name: "some_params",
	Params: tools.Parameters{
		tools.NewIntParameter("param1", "This is the first parameter."),
		tools.NewIntParameter("param2", "This is the second parameter."),
	},
}

func (t MockTool) Invoke(tools.ParamValues) ([]any, error) {
	mock := make([]any, 0)
	return mock, nil
}

// claims is a map of user info decoded from an auth token
func (t MockTool) ParseParams(data map[string]any, claimsMap map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Params, data, claimsMap)
}

func (t MockTool) Manifest() tools.Manifest {
	pMs := make([]tools.ParameterManifest, 0, len(t.Params))
	for _, p := range t.Params {
		pMs = append(pMs, p.Manifest())
	}
	return tools.Manifest{Description: t.Description, Parameters: pMs}
}

func (t MockTool) MCPTool() tools.MCPTool {
	return tools.MCPTool{Name: t.Name, Description: t.manifest.Description, InputSchema: t.manifest.ToolsSchema()}
}

func (t MockTool) Authorized(verifiedAuthServices []string) bool {
	return true
}

func TestToolsetEndpoint(t *testing.T) {
	ts, shutdown := setUpServer(t)
	defer shutdown()

	// wantResponse is a struct for checks against test cases
	type wantResponse struct {
		statusCode int
		isErr      bool
		version    string
		tools      []string
	}

	testCases := []struct {
		name        string
		toolsetName string
		want        wantResponse
	}{
		{
			name:        "'default' manifest",
			toolsetName: "",
			want: wantResponse{
				statusCode: http.StatusOK,
				version:    fakeVersionString,
				tools:      []string{tool1.Name, tool2.Name},
			},
		},
		{
			name:        "invalid toolset name",
			toolsetName: "some_imaginary_toolset",
			want: wantResponse{
				statusCode: http.StatusNotFound,
				isErr:      true,
			},
		},
		{
			name:        "single toolset 1",
			toolsetName: "tool1_only",
			want: wantResponse{
				statusCode: http.StatusOK,
				version:    fakeVersionString,
				tools:      []string{tool1.Name},
			},
		},
		{
			name:        "single toolset 2",
			toolsetName: "tool2_only",
			want: wantResponse{
				statusCode: http.StatusOK,
				version:    fakeVersionString,
				tools:      []string{tool2.Name},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body, err := testRequest(ts, http.MethodGet, fmt.Sprintf("/toolset/%s", tc.toolsetName), nil)
			if err != nil {
				t.Fatalf("unexpected error during request: %s", err)
			}

			if contentType := resp.Header.Get("Content-type"); contentType != "application/json" {
				t.Fatalf("unexpected content-type header: want %s, got %s", "application/json", contentType)
			}

			if resp.StatusCode != tc.want.statusCode {
				t.Logf("response body: %s", body)
				t.Fatalf("unexpected status code: want %d, got %d", tc.want.statusCode, resp.StatusCode)
			}
			if tc.want.isErr {
				// skip the rest of the checks if this is an error case
				return
			}
			var m tools.ToolsetManifest
			err = json.Unmarshal(body, &m)
			if err != nil {
				t.Fatalf("unable to parse ToolsetManifest: %s", err)
			}
			// Check the version is correct
			if m.ServerVersion != tc.want.version {
				t.Fatalf("unexpected ServerVersion: want %q, got %q", tc.want.version, m.ServerVersion)
			}
			// validate that the tools in the toolset are correct
			for _, name := range tc.want.tools {
				_, ok := m.ToolsManifest[name]
				if !ok {
					t.Errorf("%q tool not found in manfiest", name)
				}
			}
		})
	}
}
func TestToolGetEndpoint(t *testing.T) {
	ts, shutdown := setUpServer(t)
	defer shutdown()

	// wantResponse is a struct for checks against test cases
	type wantResponse struct {
		statusCode int
		isErr      bool
		version    string
		tools      []string
	}

	testCases := []struct {
		name     string
		toolName string
		want     wantResponse
	}{
		{
			name:     "tool1",
			toolName: tool1.Name,
			want: wantResponse{
				statusCode: http.StatusOK,
				version:    fakeVersionString,
				tools:      []string{tool1.Name},
			},
		},
		{
			name:     "tool2",
			toolName: tool2.Name,
			want: wantResponse{
				statusCode: http.StatusOK,
				version:    fakeVersionString,
				tools:      []string{tool2.Name},
			},
		},
		{
			name:     "invalid tool",
			toolName: "some_imaginary_tool",
			want: wantResponse{
				statusCode: http.StatusNotFound,
				isErr:      true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body, err := testRequest(ts, http.MethodGet, fmt.Sprintf("/tool/%s", tc.toolName), nil)
			if err != nil {
				t.Fatalf("unexpected error during request: %s", err)
			}

			if contentType := resp.Header.Get("Content-type"); contentType != "application/json" {
				t.Fatalf("unexpected content-type header: want %s, got %s", "application/json", contentType)
			}

			if resp.StatusCode != tc.want.statusCode {
				t.Logf("response body: %s", body)
				t.Fatalf("unexpected status code: want %d, got %d", tc.want.statusCode, resp.StatusCode)
			}
			if tc.want.isErr {
				// skip the rest of the checks if this is an error case
				return
			}
			var m tools.ToolsetManifest
			err = json.Unmarshal(body, &m)
			if err != nil {
				t.Fatalf("unable to parse ToolsetManifest: %s", err)
			}
			// Check the version is correct
			if m.ServerVersion != tc.want.version {
				t.Fatalf("unexpected ServerVersion: want %q, got %q", tc.want.version, m.ServerVersion)
			}
			// validate that the tools in the toolset are correct
			for _, name := range tc.want.tools {
				_, ok := m.ToolsManifest[name]
				if !ok {
					t.Errorf("%q tool not found in manfiest", name)
				}
			}
		})
	}
}

func TestMcpEndpoint(t *testing.T) {
	ts, shutdown := setUpServer(t)
	defer shutdown()

	testCases := []struct {
		name  string
		isErr bool
		body  mcp.JSONRPCRequest
		want  string
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
			want: fmt.Sprintf(`{"jsonrpc":"2.0","id":"mcp-initialize","result":{"protocolVersion":"%s","capabilities":{"tools":{"listChanged":false}},"serverInfo":{"name":"%s","version":"%s"}}}`, protocolVersion, serverName, fakeVersionString),
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
			name:  "missing method",
			isErr: true,
			body: mcp.JSONRPCRequest{
				Jsonrpc: jsonrpcVersion,
				Id:      "missing-method",
				Request: mcp.Request{},
			},
			want: `{"jsonrpc":"2.0","id":"missing-method","error":{"code":-32601,"message":"method not found"}}`,
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
			want: `{"jsonrpc":"2.0","id":"invalid-method","error":{"code":-32601,"message":"invalid method foo"}}`,
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
			want: `{"jsonrpc":"2.0","id":"invalid-jsonrpc-version","error":{"code":-32600,"message":"invalid json-rpc version"}}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqMarshal, err := json.Marshal(tc.body)
			if err != nil {
				t.Fatalf("unexpected error during marshaling of body")
			}

			resp, body, err := testRequest(ts, http.MethodPost, "/mcp", bytes.NewBuffer(reqMarshal))
			if err != nil {
				t.Fatalf("unexpected error during request: %s", err)
			}

			// Notifications don't expect a response.
			if tc.want != "" {
				if contentType := resp.Header.Get("Content-type"); contentType != "application/json" {
					t.Fatalf("unexpected content-type header: want %s, got %s", "application/json", contentType)
				}

				if diff := cmp.Diff(tc.want, string(body)); diff != "" {
					t.Fatalf("Mismatch (-want +got):\n%s\n", diff)
				}
			}
		})
	}
}

func setUpServer(t *testing.T) (*httptest.Server, func()) {
	ctx, cancel := context.WithCancel(context.Background())

	// Set up resources to test against
	toolsMap := map[string]tools.Tool{tool1.Name: tool1, tool2.Name: tool2}

	toolsets := make(map[string]tools.Toolset)
	for name, l := range map[string][]string{
		"":           {tool1.Name, tool2.Name},
		"tool1_only": {tool1.Name},
		"tool2_only": {tool2.Name},
	} {
		tc := tools.ToolsetConfig{Name: name, ToolNames: l}
		m, err := tc.Initialize(fakeVersionString, toolsMap)
		if err != nil {
			t.Fatalf("unable to initialize toolset %q: %s", name, err)
		}
		toolsets[name] = m
	}

	testLogger, err := log.NewStdLogger(os.Stdout, os.Stderr, "info")
	if err != nil {
		t.Fatalf("unable to initialize logger: %s", err)
	}

	otelShutdown, err := telemetry.SetupOTel(ctx, fakeVersionString, "", false, "toolbox")
	if err != nil {
		t.Fatalf("unable to setup otel: %s", err)
	}

	instrumentation, err := CreateTelemetryInstrumentation(fakeVersionString)
	if err != nil {
		t.Fatalf("unable to create custom metrics: %s", err)
	}

	server := Server{version: fakeVersionString, logger: testLogger, instrumentation: instrumentation, tools: toolsMap, toolsets: toolsets}
	r, err := apiRouter(&server)
	if err != nil {
		t.Fatalf("unable to initialize router: %s", err)
	}
	ts := httptest.NewServer(r)
	shutdown := func() {
		// cancel context
		cancel()
		// shutdown otel
		err := otelShutdown(ctx)
		if err != nil {
			t.Fatalf("error shutting down OpenTelemetry: %s", err)
		}
		// close server
		ts.Close()
	}
	return ts, shutdown
}

func testRequest(ts *httptest.Server, method, path string, body io.Reader) (*http.Response, []byte, error) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to send request: %w", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read request body: %w", err)
	}
	defer resp.Body.Close()

	return resp, bytes.TrimSpace(respBody), nil
}
