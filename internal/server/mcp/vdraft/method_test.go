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
	"strings"
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/server/mcp/jsonrpc"
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
