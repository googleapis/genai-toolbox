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

package util

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/auth"
	"github.com/googleapis/mcp-toolbox/internal/auth/generic"
	"github.com/googleapis/mcp-toolbox/internal/util"
)

type mockAuthService struct {
	name       string
	mcpEnabled bool
}

func (aS mockAuthService) AuthServiceType() string {
	return "mock"
}

func (aS mockAuthService) GetName() string {
	return aS.name
}

func (aS mockAuthService) GetClaimsFromHeader(context.Context, http.Header) (map[string]any, error) {
	return nil, nil
}

func (aS mockAuthService) ToConfig() auth.AuthServiceConfig {
	return generic.Config{
		Name:       aS.name,
		McpEnabled: aS.mcpEnabled,
	}
}

func TestValidateScopes(t *testing.T) {
	tests := []struct {
		name         string
		toolScopes   []string
		authServices map[string]auth.AuthService
		setupCtx     func(context.Context) context.Context
		wantErr      bool
		errMessage   string
	}{
		{
			name:       "no required scopes",
			toolScopes: nil,
			authServices: map[string]auth.AuthService{
				"svc1": mockAuthService{name: "svc1", mcpEnabled: true},
			},
			setupCtx: func(ctx context.Context) context.Context {
				return ctx
			},
			wantErr: false,
		},
		{
			name:       "mcp auth service not enabled",
			toolScopes: []string{"read:files"},
			authServices: map[string]auth.AuthService{
				"svc1": mockAuthService{name: "svc1", mcpEnabled: false},
			},
			setupCtx: func(ctx context.Context) context.Context {
				return ctx
			},
			wantErr: false,
		},
		{
			name:       "missing claims in context",
			toolScopes: []string{"read:files"},
			authServices: map[string]auth.AuthService{
				"svc1": mockAuthService{name: "svc1", mcpEnabled: true},
			},
			setupCtx: func(ctx context.Context) context.Context {
				return ctx
			},
			wantErr:    true,
			errMessage: "missing claims for MCP authorization",
		},
		{
			name:       "insufficient scopes",
			toolScopes: []string{"read:files", "write:files"},
			authServices: map[string]auth.AuthService{
				"svc1": mockAuthService{name: "svc1", mcpEnabled: true},
			},
			setupCtx: func(ctx context.Context) context.Context {
				claims := map[string]any{
					"scope": "read:files",
				}
				return util.WithAuthTokenClaims(ctx, claims)
			},
			wantErr:    true,
			errMessage: "insufficient scopes for this tool",
		},
		{
			name:       "sufficient scopes",
			toolScopes: []string{"read:files", "write:files"},
			authServices: map[string]auth.AuthService{
				"svc1": mockAuthService{name: "svc1", mcpEnabled: true},
			},
			setupCtx: func(ctx context.Context) context.Context {
				claims := map[string]any{
					"scope": "read:files write:files other:scope",
				}
				return util.WithAuthTokenClaims(ctx, claims)
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.setupCtx(context.Background())
			err := ValidateScopes(ctx, tc.toolScopes, tc.authServices)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var authErr *generic.MCPAuthError
				if !errors.As(err, &authErr) {
					t.Fatalf("expected generic.MCPAuthError, got: %T", err)
				}
				if authErr.Code != http.StatusForbidden {
					t.Errorf("expected Code StatusForbidden (403), got: %d", authErr.Code)
				}
				if authErr.Message != tc.errMessage {
					t.Errorf("expected error message %q, got %q", tc.errMessage, authErr.Message)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			}
		})
	}
}
