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
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/log"
	"github.com/googleapis/mcp-toolbox/internal/server/mcp/jsonrpc"
	"github.com/googleapis/mcp-toolbox/internal/server/resources"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
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
	// Initialize tools using provided testutils mock instances
	mockTools := []testutils.MockTool{testutils.MockTool1, testutils.MockTool2}
	toolsMap, toolsets, promptsMap, promptsets := testutils.SetUpResources(t, mockTools, nil)
	resourceMgr := resources.NewResourceManager(nil, nil, nil, toolsMap, toolsets, promptsMap, promptsets)

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
			got, err := toolsListHandler(dummyID, resourceMgr, tt.toolset, body, tt.header)

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

type mockToolWithEchoInvoke struct {
	tools.Tool
}

func (m mockToolWithEchoInvoke) Invoke(ctx context.Context, s tools.SourceProvider, params parameters.ParamValues, token tools.AccessToken) (any, util.ToolboxError) {
	val := params.AsMap()["advanced_param"]
	return val, nil
}

type mockToolWithOneOfEchoInvoke struct {
	tools.Tool
}

func (m mockToolWithOneOfEchoInvoke) Invoke(ctx context.Context, s tools.SourceProvider, params parameters.ParamValues, token tools.AccessToken) (any, util.ToolboxError) {
	val := params.AsMap()["oneof_param"]
	if val == nil {
		return "nil-value", nil
	}
	return val, nil
}

type mockCompositeParameter struct {
	*parameters.StringParameter
	manifest parameters.ParameterMcpManifest
}

func (p mockCompositeParameter) McpManifest() (parameters.ParameterMcpManifest, []string) {
	return p.manifest, nil
}

func (p mockCompositeParameter) Parse(v any) (any, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected object, got %T", v)
	}

	// allOf constraint: must contain "id" (number)
	idVal, hasId := m["id"]
	if !hasId {
		return nil, fmt.Errorf("missing required field 'id' (allOf constraint)")
	}
	switch idVal.(type) {
	case float64, int, int64, json.Number:
		// valid
	default:
		return nil, fmt.Errorf("field 'id' must be a number, got %T", idVal)
	}

	// anyOf constraint: must contain either "email" (string) or "phone" (string)
	emailVal, hasEmail := m["email"]
	phoneVal, hasPhone := m["phone"]
	if !hasEmail && !hasPhone {
		return nil, fmt.Errorf("must provide either 'email' or 'phone' (anyOf constraint)")
	}
	if hasEmail {
		if _, ok := emailVal.(string); !ok {
			return nil, fmt.Errorf("field 'email' must be a string, got %T", emailVal)
		}
	}
	if hasPhone {
		if _, ok := phoneVal.(string); !ok {
			return nil, fmt.Errorf("field 'phone' must be a string, got %T", phoneVal)
		}
	}

	return m, nil
}

type mockToolWithCompositeEchoInvoke struct {
	tools.Tool
}

func (m mockToolWithCompositeEchoInvoke) Invoke(ctx context.Context, s tools.SourceProvider, params parameters.ParamValues, token tools.AccessToken) (any, util.ToolboxError) {
	val := params.AsMap()["composite_param"]
	return val, nil
}

type mockContradictoryParameter struct {
	*parameters.StringParameter
	manifest parameters.ParameterMcpManifest
}

func (p mockContradictoryParameter) McpManifest() (parameters.ParameterMcpManifest, []string) {
	return p.manifest, nil
}

func (p mockContradictoryParameter) Parse(v any) (any, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected object, got %T", v)
	}

	// allOf constraint: must contain "id"
	_, hasId := m["id"]
	if !hasId {
		return nil, fmt.Errorf("missing required field 'id' (allOf constraint)")
	}

	// not constraint: must NOT contain "id"
	if hasId {
		return nil, fmt.Errorf("must not contain field 'id' (not constraint)")
	}

	return m, nil
}

type mockToolWithContradictoryEchoInvoke struct {
	tools.Tool
}

func (m mockToolWithContradictoryEchoInvoke) Invoke(ctx context.Context, s tools.SourceProvider, params parameters.ParamValues, token tools.AccessToken) (any, util.ToolboxError) {
	val := params.AsMap()["contradictory_param"]
	return val, nil
}

func TestToolsCallHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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

	toolsMap := make(map[string]tools.Tool)
	var allTools []string
	for _, tool := range mockTools {
		toolsMap[tool.Name] = tool
		allTools = append(allTools, tool.Name)
	}

	// Register our advanced_tool with mockAdvancedParameter
	stringParam := parameters.NewStringParameter("advanced_param", "an advanced parameter")
	advancedParam := mockAdvancedParameter{
		StringParameter: stringParam,
		manifest: parameters.ParameterMcpManifest{
			AnyOf: []*parameters.ParameterMcpManifest{
				{Type: "string"},
				{Type: "integer"},
			},
		},
	}
	advancedTool := mockToolWithEchoInvoke{
		Tool: testutils.NewMockTool("advanced_tool", "", parameters.Parameters{advancedParam}, false, false),
	}
	toolsMap["advanced_tool"] = advancedTool
	allTools = append(allTools, "advanced_tool")

	// Register our oneof_tool with mockOneOfParameter
	stringParam2 := parameters.NewStringParameterWithRequired("oneof_param", "a oneOf parameter", false)
	oneOfParam := mockOneOfParameter{
		StringParameter: stringParam2,
		manifest: parameters.ParameterMcpManifest{
			OneOf: []*parameters.ParameterMcpManifest{
				{Type: "string", Enum: []any{"red", "green", "blue"}},
				{Type: "null"},
			},
		},
	}
	oneOfTool := mockToolWithOneOfEchoInvoke{
		Tool: testutils.NewMockTool("oneof_tool", "", parameters.Parameters{oneOfParam}, false, false),
	}
	toolsMap["oneof_tool"] = oneOfTool
	allTools = append(allTools, "oneof_tool")

	// Register our composite_tool with mockCompositeParameter
	stringParam3 := parameters.NewStringParameter("composite_param", "a composite parameter")
	compositeParam := mockCompositeParameter{
		StringParameter: stringParam3,
		manifest: parameters.ParameterMcpManifest{
			Type: "object",
			AllOf: []*parameters.ParameterMcpManifest{
				{
					Properties: map[string]*parameters.ParameterMcpManifest{
						"id": {Type: "integer"},
					},
					Required: []string{"id"},
				},
			},
			AnyOf: []*parameters.ParameterMcpManifest{
				{
					Properties: map[string]*parameters.ParameterMcpManifest{
						"email": {Type: "string"},
					},
					Required: []string{"email"},
				},
				{
					Properties: map[string]*parameters.ParameterMcpManifest{
						"phone": {Type: "string"},
					},
					Required: []string{"phone"},
				},
			},
		},
	}
	compositeTool := mockToolWithCompositeEchoInvoke{
		Tool: testutils.NewMockTool("composite_tool", "", parameters.Parameters{compositeParam}, false, false),
	}
	toolsMap["composite_tool"] = compositeTool
	allTools = append(allTools, "composite_tool")

	// Register our contradictory_tool with mockContradictoryParameter
	stringParam4 := parameters.NewStringParameter("contradictory_param", "a contradictory parameter")
	contradictoryParam := mockContradictoryParameter{
		StringParameter: stringParam4,
		manifest: parameters.ParameterMcpManifest{
			Type: "object",
			AllOf: []*parameters.ParameterMcpManifest{
				{
					Properties: map[string]*parameters.ParameterMcpManifest{
						"id": {Type: "integer"},
					},
					Required: []string{"id"},
				},
			},
			Not: &parameters.ParameterMcpManifest{
				Properties: map[string]*parameters.ParameterMcpManifest{
					"id": {Type: "integer"},
				},
				Required: []string{"id"},
			},
		},
	}
	contradictoryTool := mockToolWithContradictoryEchoInvoke{
		Tool: testutils.NewMockTool("contradictory_tool", "", parameters.Parameters{contradictoryParam}, false, false),
	}
	toolsMap["contradictory_tool"] = contradictoryTool
	allTools = append(allTools, "contradictory_tool")

	toolsets := make(map[string]tools.Toolset)
	tc := tools.ToolsetConfig{Name: "", ToolNames: allTools}
	ts, err := tc.Initialize("test-version", toolsMap)
	if err != nil {
		t.Fatalf("unable to initialize toolset: %s", err)
	}
	toolsets[""] = ts

	resourceMgr := resources.NewResourceManager(nil, nil, nil, toolsMap, toolsets, nil, nil)

	tests := []struct {
		name        string
		body        CallToolRequest
		rawBody     []byte
		header      http.Header
		context     context.Context
		wantErr     bool
		errContains string
		wantContent []TextContent
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
		{
			name: "successful invocation - advanced_tool with string",
			body: CallToolRequest{
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: CallToolRequestParams{
					Name: "advanced_tool",
					Arguments: map[string]any{
						"advanced_param": "hello",
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
			header:      http.Header{"Mcp-Method": []string{TOOLS_CALL}, "Mcp-Name": []string{"advanced_tool"}},
			context:     ctxLogger,
			wantErr:     false,
			wantContent: []TextContent{{Type: "text", Text: `"hello"`}},
		},
		{
			name: "successful invocation - advanced_tool with number",
			body: CallToolRequest{
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: CallToolRequestParams{
					Name: "advanced_tool",
					Arguments: map[string]any{
						"advanced_param": 42.0,
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
			header:      http.Header{"Mcp-Method": []string{TOOLS_CALL}, "Mcp-Name": []string{"advanced_tool"}},
			context:     ctxLogger,
			wantErr:     false,
			wantContent: []TextContent{{Type: "text", Text: `42`}},
		},
		{
			name: "failed invocation - advanced_tool with invalid boolean type",
			body: CallToolRequest{
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: CallToolRequestParams{
					Name: "advanced_tool",
					Arguments: map[string]any{
						"advanced_param": true,
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
			header:      http.Header{"Mcp-Method": []string{TOOLS_CALL}, "Mcp-Name": []string{"advanced_tool"}},
			context:     ctxLogger,
			wantErr:     true,
			errContains: "neither a string nor a number",
		},
		{
			name: "successful invocation - oneof_tool with valid enum color",
			body: CallToolRequest{
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: CallToolRequestParams{
					Name: "oneof_tool",
					Arguments: map[string]any{
						"oneof_param": "red",
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
			header:      http.Header{"Mcp-Method": []string{TOOLS_CALL}, "Mcp-Name": []string{"oneof_tool"}},
			context:     ctxLogger,
			wantErr:     false,
			wantContent: []TextContent{{Type: "text", Text: `"red"`}},
		},
		{
			name: "successful invocation - oneof_tool with null value (nil)",
			body: CallToolRequest{
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: CallToolRequestParams{
					Name: "oneof_tool",
					Arguments: map[string]any{
						"oneof_param": nil,
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
			header:      http.Header{"Mcp-Method": []string{TOOLS_CALL}, "Mcp-Name": []string{"oneof_tool"}},
			context:     ctxLogger,
			wantErr:     false,
			wantContent: []TextContent{{Type: "text", Text: `"nil-value"`}},
		},
		{
			name: "failed invocation - oneof_tool with invalid enum color",
			body: CallToolRequest{
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: CallToolRequestParams{
					Name: "oneof_tool",
					Arguments: map[string]any{
						"oneof_param": "yellow",
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
			header:      http.Header{"Mcp-Method": []string{TOOLS_CALL}, "Mcp-Name": []string{"oneof_tool"}},
			context:     ctxLogger,
			wantErr:     true,
			errContains: "value \"yellow\" is not one of the allowed colors",
		},
		{
			name: "failed invocation - oneof_tool with invalid type (integer)",
			body: CallToolRequest{
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: CallToolRequestParams{
					Name: "oneof_tool",
					Arguments: map[string]any{
						"oneof_param": 123,
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
			header:      http.Header{"Mcp-Method": []string{TOOLS_CALL}, "Mcp-Name": []string{"oneof_tool"}},
			context:     ctxLogger,
			wantErr:     true,
			errContains: "neither a valid color nor null",
		},
		{
			name: "successful invocation - composite_tool with id and email",
			body: CallToolRequest{
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: CallToolRequestParams{
					Name: "composite_tool",
					Arguments: map[string]any{
						"composite_param": map[string]any{
							"id":    float64(123),
							"email": "test@example.com",
						},
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
			header:      http.Header{"Mcp-Method": []string{TOOLS_CALL}, "Mcp-Name": []string{"composite_tool"}},
			context:     ctxLogger,
			wantErr:     false,
			wantContent: []TextContent{{Type: "text", Text: `{"email":"test@example.com","id":123}`}},
		},
		{
			name: "failed invocation - composite_tool missing id (fails allOf)",
			body: CallToolRequest{
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: CallToolRequestParams{
					Name: "composite_tool",
					Arguments: map[string]any{
						"composite_param": map[string]any{
							"email": "test@example.com",
						},
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
			header:      http.Header{"Mcp-Method": []string{TOOLS_CALL}, "Mcp-Name": []string{"composite_tool"}},
			context:     ctxLogger,
			wantErr:     true,
			errContains: "missing required field 'id' (allOf constraint)",
		},
		{
			name: "failed invocation - composite_tool missing email and phone (fails anyOf)",
			body: CallToolRequest{
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: CallToolRequestParams{
					Name: "composite_tool",
					Arguments: map[string]any{
						"composite_param": map[string]any{
							"id": float64(123),
						},
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
			header:      http.Header{"Mcp-Method": []string{TOOLS_CALL}, "Mcp-Name": []string{"composite_tool"}},
			context:     ctxLogger,
			wantErr:     true,
			errContains: "must provide either 'email' or 'phone' (anyOf constraint)",
		},
		{
			name: "failed invocation - contradictory_tool missing id (fails allOf)",
			body: CallToolRequest{
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: CallToolRequestParams{
					Name: "contradictory_tool",
					Arguments: map[string]any{
						"contradictory_param": map[string]any{},
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
			header:      http.Header{"Mcp-Method": []string{TOOLS_CALL}, "Mcp-Name": []string{"contradictory_tool"}},
			context:     ctxLogger,
			wantErr:     true,
			errContains: "missing required field 'id' (allOf constraint)",
		},
		{
			name: "failed invocation - contradictory_tool has id (fails not)",
			body: CallToolRequest{
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: CallToolRequestParams{
					Name: "contradictory_tool",
					Arguments: map[string]any{
						"contradictory_param": map[string]any{
							"id": float64(123),
						},
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
			header:      http.Header{"Mcp-Method": []string{TOOLS_CALL}, "Mcp-Name": []string{"contradictory_tool"}},
			context:     ctxLogger,
			wantErr:     true,
			errContains: "must not contain field 'id' (not constraint)",
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
			got, err := toolsCallHandler(tt.context, dummyID, toolsets[""], resourceMgr, body, tt.header)

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
				if tt.wantContent != nil {
					resp, ok := got.(jsonrpc.JSONRPCResponse)
					if !ok {
						t.Fatalf("expected jsonrpc.JSONRPCResponse, got %T", got)
					}
					result, ok := resp.Result.(CallToolResult)
					if !ok {
						t.Fatalf("expected CallToolResult, got %T", resp.Result)
					}
					if diff := cmp.Diff(result.Content, tt.wantContent); diff != "" {
						t.Errorf("unexpected Content (-want +got):\n%s", diff)
					}
				}
			}
		})
	}
}

func TestPromptsListHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	testLogger, err := log.NewStdLogger(os.Stdout, os.Stderr, "info")
	if err != nil {
		t.Fatalf("unable to initialize logger: %s", err)
	}
	ctx = util.WithLogger(ctx, testLogger)
	// Initialize prompts
	mockPrompts := []testutils.MockPrompt{testutils.MockPrompt1, testutils.MockPrompt2}
	toolsMap, toolsets, promptsMap, promptsets := testutils.SetUpResources(t, nil, mockPrompts)
	resourceMgr := resources.NewResourceManager(nil, nil, nil, toolsMap, toolsets, promptsMap, promptsets)
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
			got, err := promptsListHandler(ctx, dummyID, resourceMgr, promptsets[""], body, tt.header)

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
	testLogger, err := log.NewStdLogger(os.Stdout, os.Stderr, "info")
	if err != nil {
		t.Fatalf("unable to initialize logger: %s", err)
	}
	ctx = util.WithLogger(ctx, testLogger)
	// Initialize prompts
	mockPrompts := []testutils.MockPrompt{testutils.MockPrompt1, testutils.MockPrompt2}
	toolsMap, toolsets, promptsMap, promptsets := testutils.SetUpResources(t, nil, mockPrompts)
	resourceMgr := resources.NewResourceManager(nil, nil, nil, toolsMap, toolsets, promptsMap, promptsets)
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
			got, err := promptsGetHandler(ctx, dummyID, promptsets[""], resourceMgr, body, tt.header)

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
