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

package vdraft

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/log"
	"github.com/googleapis/mcp-toolbox/internal/server/mcp/jsonrpc"
	"github.com/googleapis/mcp-toolbox/internal/server/primitives"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
)

// Dummy JSONRPC ID for testing
var (
	dummyID           jsonrpc.RequestId = 1
	fakeVersionString                   = "0.0.0"
)

func TestValidateMetadata(t *testing.T) {
	var dummyId jsonrpc.RequestId
	clientCapabilities := &ClientCapabilities{}

	tests := []struct {
		name        string
		params      RequestParams
		stdio       bool
		wantErr     bool
		errContains string
	}{
		{
			name: "Missing Meta entirely",
			params: RequestParams{
				Meta: nil,
			},
			stdio:       true,
			wantErr:     true,
			errContains: "missing required fields in request metadata",
		},
		{
			name: "Missing Protocol Version",
			params: RequestParams{
				Meta: &RequestMetaObject{}, // ProtocolVersion defaults to ""
			},
			stdio:       true,
			wantErr:     true,
			errContains: "missing io.modelcontextprotocol/protocolVersion",
		},
		{
			name: "Protocol Version Mismatch (non-stdio)",
			params: RequestParams{
				Meta: &RequestMetaObject{
					ProtocolVersion: "invalid-version-999",
				},
			},
			stdio:       false,
			wantErr:     true,
			errContains: "header mismatch",
		},
		{
			name: "Missing ClientInfo Name",
			params: RequestParams{
				Meta: &RequestMetaObject{
					ProtocolVersion: PROTOCOL_VERSION,
					ClientInfo: Implementation{
						Version:      "1.0",
						BaseMetadata: BaseMetadata{Name: ""}, // Missing name
					},
				},
			},
			stdio:       true,
			wantErr:     true,
			errContains: "missing field from io.modelcontextprotocol/clientInfo",
		},
		{
			name: "Missing ClientInfo Version",
			params: RequestParams{
				Meta: &RequestMetaObject{
					ProtocolVersion: PROTOCOL_VERSION,
					ClientInfo: Implementation{
						BaseMetadata: BaseMetadata{Name: "TestClient"},
						Version:      "", // Missing version
					},
				},
			},
			stdio:       true,
			wantErr:     true,
			errContains: "missing field from io.modelcontextprotocol/clientInfo",
		},
		{
			name: "Missing Client Capabilities",
			params: RequestParams{
				Meta: &RequestMetaObject{
					ProtocolVersion: PROTOCOL_VERSION,
					ClientInfo: Implementation{
						BaseMetadata: BaseMetadata{Name: "TestClient"},
						Version:      "1.0",
					},
					MetaClientCapabilities: nil, // Missing capabilities
				},
			},
			stdio:       true,
			wantErr:     true,
			errContains: "missing field from io.modelcontextprotocol/clientCapabilities",
		},
		{
			name: "stdio transport",
			params: RequestParams{
				Meta: &RequestMetaObject{
					// ProtocolVersion can be anything if stdio is true
					// Technically it will be valid and would already be
					// verified during message processing
					ProtocolVersion: "any-version",
					ClientInfo: Implementation{
						BaseMetadata: BaseMetadata{Name: "TestClient"},
						Version:      "1.0",
					},
					MetaClientCapabilities: clientCapabilities,
				},
			},
			stdio:   true,
			wantErr: false,
		},
		{
			name: "Success request metadata",
			params: RequestParams{
				Meta: &RequestMetaObject{
					ProtocolVersion: PROTOCOL_VERSION, // Must match exactly when stdio=false
					ClientInfo: Implementation{
						BaseMetadata: BaseMetadata{Name: "TestClient"},
						Version:      "1.0",
					},
					MetaClientCapabilities: clientCapabilities,
				},
			},
			stdio:   false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := validateMetadata(dummyId, tt.params, tt.stdio)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("validateMetadata() expected an error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateMetadata() error = %v, want error containing %q", err, tt.errContains)
				}
				if res == nil {
					t.Errorf("validateMetadata() expected jsonrpc error response, got nil res")
				}
			} else {
				if err != nil {
					t.Errorf("validateMetadata() expected no error, got %v", err)
				}
				if res != nil {
					t.Errorf("validateMetadata() expected nil res on success, got %v", res)
				}
			}
		})
	}
}

func TestValidateHeader(t *testing.T) {
	tests := []struct {
		name    string
		header  http.Header
		method  string
		reqName string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil header (stdio transport)",
			header:  nil,
			method:  "test-method",
			reqName: "test-name",
			wantErr: false,
		},
		{
			name: "valid header matches body",
			header: http.Header{
				"Mcp-Method": []string{"test-method"},
				"Mcp-Name":   []string{"test-name"},
			},
			method:  "test-method",
			reqName: "test-name",
			wantErr: false,
		},
		{
			name: "mismatched method",
			header: http.Header{
				"Mcp-Method": []string{"wrong-method"},
				"Mcp-Name":   []string{"test-name"},
			},
			method:  "test-method",
			reqName: "test-name",
			wantErr: true,
			errMsg:  "Mcp-Method header value 'wrong-method' does not match body value 'test-method'",
		},
		{
			name: "mismatched name",
			header: http.Header{
				"Mcp-Method": []string{"test-method"},
				"Mcp-Name":   []string{"wrong-name"},
			},
			method:  "test-method",
			reqName: "test-name",
			wantErr: true,
			errMsg:  "Mcp-Name header value 'wrong-name' does not match body value 'test-name'",
		},
		{
			name: "missing method in header",
			header: http.Header{
				"Mcp-Name": []string{"test-name"},
			},
			method:  "test-method",
			reqName: "test-name",
			wantErr: true,
			errMsg:  "Mcp-Method header value '' does not match body value 'test-method'",
		},
		{
			name: "missing name in header",
			header: http.Header{
				"Mcp-Method": []string{"test-method"},
			},
			method:  "test-method",
			reqName: "test-name",
			wantErr: true,
			errMsg:  "Mcp-Name header value '' does not match body value 'test-name'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotObj, err := validateHeader(dummyID, tt.header, tt.method, tt.reqName)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("validateHeader() expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateHeader() error = %v, wantMsg %v", err, tt.errMsg)
				}
				if gotObj == nil {
					t.Errorf("validateHeader() expected an error object return value, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("validateHeader() unexpected error: %v", err)
				}
				if gotObj != nil {
					t.Errorf("validateHeader() expected nil object, got %v", gotObj)
				}
			}
		})
	}
}

func TestServerDiscoverHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = util.WithEnableDraftSpecs(ctx, true)
	defer cancel()
	ctxVersion := util.WithToolboxVersionKey(ctx, fakeVersionString)
	tests := []struct {
		name        string
		body        DiscoverRequest
		rawBody     []byte
		header      http.Header
		context     context.Context
		wantErr     bool
		errContains string
	}{
		{
			name: "missing version in context",
			body: DiscoverRequest{
				Request: jsonrpc.Request{
					Method: "server/discover",
				},
				Params: RequestParams{
					Meta: &RequestMetaObject{
						ProtocolVersion: PROTOCOL_VERSION,
						ClientInfo: Implementation{
							BaseMetadata: BaseMetadata{Name: "TestClient"},
							Version:      "1.0",
						},
						MetaClientCapabilities: &ClientCapabilities{},
					},
				},
			},
			header:      nil,
			context:     ctx,
			wantErr:     true,
			errContains: "unable to retrieve toolbox version",
		},
		{
			name:        "invalid json body",
			rawBody:     []byte(`{invalid json}`),
			header:      nil,
			context:     ctxVersion,
			wantErr:     true,
			errContains: "invalid server discover request",
		},
		{
			name: "header validation failure",
			body: DiscoverRequest{
				Request: jsonrpc.Request{
					Method: "server/discover",
				},
				Params: RequestParams{
					Meta: &RequestMetaObject{
						ProtocolVersion: PROTOCOL_VERSION,
						ClientInfo: Implementation{
							BaseMetadata: BaseMetadata{Name: "TestClient"},
							Version:      "1.0",
						},
						MetaClientCapabilities: &ClientCapabilities{},
					},
				},
			},
			header:      http.Header{"Mcp-Method": []string{"WRONG_METHOD"}},
			context:     ctxVersion,
			wantErr:     true,
			errContains: "does not match body value",
		},
		{
			name: "success",
			body: DiscoverRequest{
				Request: jsonrpc.Request{
					Method: "server/discover",
				},
				Params: RequestParams{
					Meta: &RequestMetaObject{
						ProtocolVersion: PROTOCOL_VERSION,
						ClientInfo: Implementation{
							BaseMetadata: BaseMetadata{Name: "TestClient"},
							Version:      "1.0",
						},
						MetaClientCapabilities: &ClientCapabilities{},
					},
				},
			},
			header:  http.Header{"Mcp-Method": []string{SERVER_DISCOVER}},
			context: ctxVersion,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tt.rawBody
			var err error
			if body == nil {
				body, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("unexpected error during marshaling")
				}
			}
			got, err := serverDiscoverHandler(tt.context, dummyID, body, tt.header)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got == nil {
					t.Errorf("expected valid response, got nil")
				}
			}
		})
	}
}

func TestToolsListHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = util.WithToolboxVersionKey(ctx, "v0.0.0")
	// Initialize tools using provided testutils mock instances
	mockTools := []testutils.MockTool{testutils.MockTool1, testutils.MockTool2}
	toolsMap, toolsets, promptsMap, promptsets := testutils.SetUpResources(t, mockTools, nil)
	primitiveMgr := primitives.NewPrimitiveManager(nil, nil, nil, toolsMap, toolsets, promptsMap, promptsets)

	tests := []struct {
		name        string
		body        ListToolsRequest
		rawBody     []byte
		header      http.Header
		toolset     tools.Toolset
		wantErr     bool
		errContains string
	}{
		{
			name:        "invalid json body",
			rawBody:     []byte(`{invalid json}`),
			header:      nil,
			toolset:     toolsets[""],
			wantErr:     true,
			errContains: "invalid mcp tools list request",
		},
		{
			name: "header mismatch",
			body: ListToolsRequest{
				PaginatedRequest: PaginatedRequest{
					Request: jsonrpc.Request{
						Method: "tools/list",
					},
					Params: PaginatedRequestParams{
						RequestParams: RequestParams{
							Meta: &RequestMetaObject{
								ProtocolVersion: PROTOCOL_VERSION,
								ClientInfo: Implementation{
									BaseMetadata: BaseMetadata{Name: "TestClient"},
									Version:      "1.0",
								},
								MetaClientCapabilities: &ClientCapabilities{},
							},
						},
					},
				},
			},
			header:      http.Header{"Mcp-Method": []string{"WRONG_METHOD"}},
			toolset:     toolsets[""],
			wantErr:     true,
			errContains: "does not match body value",
		},
		{
			name: "success - stdio (nil header)",
			body: ListToolsRequest{
				PaginatedRequest: PaginatedRequest{
					Request: jsonrpc.Request{
						Method: "tools/list",
					},
					Params: PaginatedRequestParams{
						RequestParams: RequestParams{
							Meta: &RequestMetaObject{
								ProtocolVersion: PROTOCOL_VERSION,
								ClientInfo: Implementation{
									BaseMetadata: BaseMetadata{Name: "TestClient"},
									Version:      "1.0",
								},
								MetaClientCapabilities: &ClientCapabilities{},
							},
						},
					},
				},
			},
			header:  nil,
			toolset: toolsets[""],
			wantErr: false,
		},
		{
			name: "success - http",
			body: ListToolsRequest{
				PaginatedRequest: PaginatedRequest{
					Request: jsonrpc.Request{
						Method: "tools/list",
					},
					Params: PaginatedRequestParams{
						RequestParams: RequestParams{
							Meta: &RequestMetaObject{
								ProtocolVersion: PROTOCOL_VERSION,
								ClientInfo: Implementation{
									BaseMetadata: BaseMetadata{Name: "TestClient"},
									Version:      "1.0",
								},
								MetaClientCapabilities: &ClientCapabilities{},
							},
						},
					},
				},
			},
			header:  http.Header{"Mcp-Method": []string{TOOLS_LIST}},
			toolset: toolsets[""],
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tt.rawBody
			var err error
			if body == nil {
				body, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("unexpected error during marshaling")
				}
			}
			got, err := toolsListHandler(ctx, dummyID, primitiveMgr, tt.toolset, body, tt.header)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want string containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got == nil {
					t.Errorf("expected valid response, got nil")
				}
			}
		})
	}
}

func TestToolsCallHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = util.WithToolboxVersionKey(ctx, "v0.0.0")
	testLogger, err := log.NewStdLogger(os.Stdout, os.Stderr, "info")
	if err != nil {
		t.Fatalf("unable to initialize logger: %s", err)
	}
	ctxLogger := util.WithLogger(ctx, testLogger)
	// Setup tools including the auth/unauth ones
	mockTools := []testutils.MockTool{
		testutils.MockTool1,
		testutils.MockTool4,
		testutils.MockTool5,
	}
	toolsMap, toolsets, promptsMap, promptsets := testutils.SetUpResources(t, mockTools, nil)
	primitiveMgr := primitives.NewPrimitiveManager(nil, nil, nil, toolsMap, toolsets, promptsMap, promptsets)

	tests := []struct {
		name        string
		body        CallToolRequest
		rawBody     []byte
		header      http.Header
		context     context.Context
		wantErr     bool
		errContains string
	}{
		{
			name:        "invalid json body",
			rawBody:     []byte(`{invalid json}`),
			header:      nil,
			context:     ctxLogger,
			wantErr:     true,
			errContains: "invalid mcp tools call request",
		},
		{
			name: "missing logger in context",
			body: CallToolRequest{
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: CallToolRequestParams{
					Name: "no_params",
					RequestParams: RequestParams{
						Meta: &RequestMetaObject{
							ProtocolVersion: PROTOCOL_VERSION,
							ClientInfo: Implementation{
								BaseMetadata: BaseMetadata{Name: "TestClient"},
								Version:      "1.0",
							},
							MetaClientCapabilities: &ClientCapabilities{},
						},
					},
				},
			},
			header:      nil,
			context:     ctx,
			wantErr:     true,
			errContains: "unable to retrieve logger",
		},
		{
			name: "tool not in toolset",
			body: CallToolRequest{
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: CallToolRequestParams{
					Name: "unknown_tool",
					RequestParams: RequestParams{
						Meta: &RequestMetaObject{
							ProtocolVersion: PROTOCOL_VERSION,
							ClientInfo: Implementation{
								BaseMetadata: BaseMetadata{Name: "TestClient"},
								Version:      "1.0",
							},
							MetaClientCapabilities: &ClientCapabilities{},
						},
					},
				},
			},
			header:      nil,
			context:     ctxLogger,
			wantErr:     true,
			errContains: "tool with name \"unknown_tool\" does not exist",
		},
		{
			name: "missing client auth token",
			body: CallToolRequest{
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: CallToolRequestParams{
					Name: "require_client_auth_tool",
					RequestParams: RequestParams{
						Meta: &RequestMetaObject{
							ProtocolVersion: PROTOCOL_VERSION,
							ClientInfo: Implementation{
								BaseMetadata: BaseMetadata{Name: "TestClient"},
								Version:      "1.0",
							},
							MetaClientCapabilities: &ClientCapabilities{},
						},
					},
				},
			},
			header:      http.Header{"Mcp-Method": []string{TOOLS_CALL}, "Mcp-Name": []string{"require_client_auth_tool"}},
			context:     ctxLogger,
			wantErr:     true,
			errContains: "missing access token in the 'Authorization' header",
		},
		{
			name: "successful invocation - no params",
			body: CallToolRequest{
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: CallToolRequestParams{
					Name: "no_params",
					RequestParams: RequestParams{
						Meta: &RequestMetaObject{
							ProtocolVersion: PROTOCOL_VERSION,
							ClientInfo: Implementation{
								BaseMetadata: BaseMetadata{Name: "TestClient"},
								Version:      "1.0",
							},
							MetaClientCapabilities: &ClientCapabilities{},
						},
					},
				},
			},
			header:  http.Header{"Mcp-Method": []string{TOOLS_CALL}, "Mcp-Name": []string{"no_params"}},
			context: ctxLogger,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tt.rawBody
			var err error
			if body == nil {
				body, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("unexpected error during marshaling")
				}
			}
			got, err := toolsCallHandler(tt.context, dummyID, toolsets[""], primitiveMgr, body, tt.header)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want string containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got == nil {
					t.Errorf("expected valid response, got nil")
				}
			}
		})
	}
}

func TestPromptsListHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = util.WithToolboxVersionKey(ctx, "v0.0.0")
	testLogger, err := log.NewStdLogger(os.Stdout, os.Stderr, "info")
	if err != nil {
		t.Fatalf("unable to initialize logger: %s", err)
	}
	ctx = util.WithLogger(ctx, testLogger)
	// Initialize prompts
	mockPrompts := []testutils.MockPrompt{testutils.MockPrompt1, testutils.MockPrompt2}
	toolsMap, toolsets, promptsMap, promptsets := testutils.SetUpResources(t, nil, mockPrompts)
	primitiveMgr := primitives.NewPrimitiveManager(nil, nil, nil, toolsMap, toolsets, promptsMap, promptsets)
	tests := []struct {
		name        string
		body        ListPromptsRequest
		rawBody     []byte
		header      http.Header
		wantErr     bool
		errContains string
	}{
		{
			name:        "invalid json request",
			rawBody:     []byte(`{invalid json}`),
			header:      nil,
			wantErr:     true,
			errContains: "invalid mcp prompts list request",
		},
		{
			name: "success",
			body: ListPromptsRequest{
				PaginatedRequest: PaginatedRequest{
					Request: jsonrpc.Request{
						Method: "prompts/list",
					},
					Params: PaginatedRequestParams{
						RequestParams: RequestParams{
							Meta: &RequestMetaObject{
								ProtocolVersion: PROTOCOL_VERSION,
								ClientInfo: Implementation{
									BaseMetadata: BaseMetadata{Name: "TestClient"},
									Version:      "1.0",
								},
								MetaClientCapabilities: &ClientCapabilities{},
							},
						},
					},
				},
			},
			header:  http.Header{"Mcp-Method": []string{PROMPTS_LIST}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tt.rawBody
			var err error
			if body == nil {
				body, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("unexpected error during marshaling")
				}
			}
			got, err := promptsListHandler(ctx, dummyID, primitiveMgr, promptsets[""], body, tt.header)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want string containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got == nil {
					t.Errorf("expected valid response, got nil")
				}
			}
		})
	}
}

func TestPromptsGetHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = util.WithToolboxVersionKey(ctx, "v0.0.0")
	testLogger, err := log.NewStdLogger(os.Stdout, os.Stderr, "info")
	if err != nil {
		t.Fatalf("unable to initialize logger: %s", err)
	}
	ctx = util.WithLogger(ctx, testLogger)
	// Initialize prompts
	mockPrompts := []testutils.MockPrompt{testutils.MockPrompt1, testutils.MockPrompt2}
	toolsMap, toolsets, promptsMap, promptsets := testutils.SetUpResources(t, nil, mockPrompts)
	primitiveMgr := primitives.NewPrimitiveManager(nil, nil, nil, toolsMap, toolsets, promptsMap, promptsets)
	tests := []struct {
		name        string
		body        GetPromptRequest
		rawBody     []byte
		header      http.Header
		wantErr     bool
		errContains string
	}{
		{
			name:        "invalid json request",
			rawBody:     []byte(`{invalid json}`),
			header:      nil,
			wantErr:     true,
			errContains: "invalid mcp prompts/get request",
		},
		{
			name: "prompt does not exist",
			body: GetPromptRequest{
				Request: jsonrpc.Request{
					Method: "prompts/get",
				},
				Params: GetPromptRequestParams{
					Name: "missing_prompt",
					RequestParams: RequestParams{
						Meta: &RequestMetaObject{
							ProtocolVersion: PROTOCOL_VERSION,
							ClientInfo: Implementation{
								BaseMetadata: BaseMetadata{Name: "TestClient"},
								Version:      "1.0",
							},
							MetaClientCapabilities: &ClientCapabilities{},
						},
					},
				},
			},
			header:      nil,
			wantErr:     true,
			errContains: "does not exist",
		},
		{
			name: "success with args",
			body: GetPromptRequest{
				Request: jsonrpc.Request{
					Method: "prompts/get",
				},
				Params: GetPromptRequestParams{
					Name: "prompt2",
					Arguments: map[string]any{
						"arg1": "value1",
					},
					RequestParams: RequestParams{
						Meta: &RequestMetaObject{
							ProtocolVersion: PROTOCOL_VERSION,
							ClientInfo: Implementation{
								BaseMetadata: BaseMetadata{Name: "TestClient"},
								Version:      "1.0",
							},
							MetaClientCapabilities: &ClientCapabilities{},
						},
					},
				},
			},
			header:  http.Header{"Mcp-Method": []string{PROMPTS_GET}, "Mcp-Name": []string{"prompt2"}},
			wantErr: false,
		},
		{
			name: "success without args",
			body: GetPromptRequest{
				Request: jsonrpc.Request{
					Method: "prompts/get",
				},
				Params: GetPromptRequestParams{
					Name: "prompt1",
					RequestParams: RequestParams{
						Meta: &RequestMetaObject{
							ProtocolVersion: PROTOCOL_VERSION,
							ClientInfo: Implementation{
								BaseMetadata: BaseMetadata{Name: "TestClient"},
								Version:      "1.0",
							},
							MetaClientCapabilities: &ClientCapabilities{},
						},
					},
				},
			},
			header:  http.Header{"Mcp-Method": []string{PROMPTS_GET}, "Mcp-Name": []string{"prompt1"}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tt.rawBody
			var err error
			if body == nil {
				body, err = json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("unexpected error during marshaling")
				}
			}
			got, err := promptsGetHandler(ctx, dummyID, promptsets[""], primitiveMgr, body, tt.header)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want string containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got == nil {
					t.Errorf("expected valid response, got nil")
				}
			}
		})
	}
}

func TestGetResultMetadata(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctxWithVersion := util.WithToolboxVersionKey(ctx, "v0.0.0")
	server_name := "Toolbox"
	// Define the table structure for our test cases
	tests := []struct {
		name       string
		ctx        context.Context
		curMeta    map[string]any
		want       map[string]any
		wantErr    bool
		errMessage string // Optional: check for specific error messages
	}{
		{
			name: "Success - Merge with existing metadata",
			ctx:  ctxWithVersion,
			curMeta: map[string]any{
				"existing_key": "existing_value",
				"another_key":  123,
			},
			want: map[string]any{
				"existing_key": "existing_value",
				"another_key":  123,
				"io.modelcontextprotocol/serverInfo": map[string]any{
					"name":    server_name,
					"version": "v0.0.0",
				},
			},
			wantErr: false,
		},
		{
			name:    "Success - Nil current metadata",
			ctx:     ctxWithVersion,
			curMeta: nil,
			want: map[string]any{
				"io.modelcontextprotocol/serverInfo": map[string]any{
					"name":    server_name,
					"version": "v0.0.0",
				},
			},
			wantErr: false,
		},
		{
			name: "Success - Overwrites duplicate keys in metadata",
			ctx:  ctxWithVersion,
			curMeta: map[string]any{
				"io.modelcontextprotocol/serverInfo": "old_data",
				"other_key":                          true,
			},
			want: map[string]any{
				"other_key": true,
				"io.modelcontextprotocol/serverInfo": map[string]any{
					"name":    server_name,
					"version": "v0.0.0",
				},
			},
			wantErr: false,
		},
		{
			name: "Failure - Context error (version retrieval fails)",
			ctx:  context.Background(),
			curMeta: map[string]any{
				"some_data": "value",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.ctx
			got, err := getResultMetadata(ctx, tt.curMeta)
			if (err != nil) != tt.wantErr {
				t.Fatalf("getResultMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getResultMetadata() got =\n%v\nwant =\n%v", got, tt.want)
			}
		})
	}
}
