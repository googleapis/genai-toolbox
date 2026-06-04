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
	"net/http"
	"slices"
	"strings"

	"github.com/googleapis/mcp-toolbox/internal/auth"
	"github.com/googleapis/mcp-toolbox/internal/util"
)

// ValidateScopes validates if the claims contain all required scopes for a tool call.
func ValidateScopes(ctx context.Context, toolScopes []string, authServices map[string]auth.AuthService) error {
	// Find MCP enabled auth service
	var mcpEnabled bool
	for _, aS := range authServices {
		if mSvc, ok := aS.(auth.MCPAuthService); ok && mSvc.IsMCPEnabled() {
			mcpEnabled = true
			break
		}
	}

	if mcpEnabled && len(toolScopes) > 0 {
		claims := util.AuthTokenClaimsFromContext(ctx)
		if claims == nil {
			return &auth.MCPAuthError{
				Code:           http.StatusForbidden,
				Message:        "missing claims for MCP authorization",
				ScopesRequired: toolScopes,
			}
		}

		scopeClaim, _ := claims["scope"].(string)
		tokenScopes := strings.Fields(scopeClaim)

		// Check if all required scopes are present in the token
		for _, ts := range toolScopes {
			if !slices.Contains(tokenScopes, ts) {
				return &auth.MCPAuthError{
					Code:           http.StatusForbidden,
					Message:        "insufficient scopes for this tool",
					ScopesRequired: toolScopes,
				}
			}
		}
	}

	return nil
}
