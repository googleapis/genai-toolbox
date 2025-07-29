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

package tools

import (
	"context"
	"fmt"
	"slices"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
)

// ToolConfigFactory defines the signature for a function that creates and
// decodes a specific tool's configuration. It takes the context, the tool's
// name, and a YAML decoder to parse the config.
type ToolConfigFactory func(ctx context.Context, name string, decoder *yaml.Decoder) (ToolConfig, error)

var toolRegistry = make(map[string]ToolConfigFactory)

// Register allows individual tool packages to register their configuration
// factory function. This is typically called from an init() function in the
// tool's package. It associates a 'kind' string with a function that can
// produce the specific ToolConfig type. It returns true if the registration was
// successful, and false if a tool with the same kind was already registered.
func Register(kind string, factory ToolConfigFactory) bool {
	if _, exists := toolRegistry[kind]; exists {
		// Tool with this kind already exists, do not overwrite.
		return false
	}
	toolRegistry[kind] = factory
	return true
}

// DecodeConfig looks up the registered factory for the given kind and uses it
// to decode the tool configuration.
func DecodeConfig(ctx context.Context, kind string, name string, decoder *yaml.Decoder) (ToolConfig, error) {
	factory, found := toolRegistry[kind]
	if !found {
		return nil, fmt.Errorf("unknown tool kind: %q", kind)
	}
	toolConfig, err := factory(ctx, name, decoder)
	if err != nil {
		return nil, fmt.Errorf("unable to parse tool %q as kind %q: %w", name, kind, err)
	}
	return toolConfig, nil
}

type ToolConfig interface {
	ToolConfigKind() string
	Initialize(map[string]sources.Source) (Tool, error)
}

type Tool interface {
	Invoke(context.Context, ParamValues) ([]any, error)
	ParseParams(map[string]any, map[string]map[string]any) (ParamValues, error)
	Manifest() Manifest
	McpManifest() McpManifest
	Authorized([]string) bool
}

// Manifest is the representation of tools sent to Client SDKs.
type Manifest struct {
	Description  string              `json:"description"`
	Parameters   []ParameterManifest `json:"parameters"`
	AuthRequired []string            `json:"authRequired"`
}

// Definition for a tool the MCP client can call.
type McpManifest struct {
	// The name of the tool.
	Name string `json:"name"`
	// A human-readable description of the tool.
	Description string `json:"description,omitempty"`
	// A JSON Schema object defining the expected parameters for the tool.
	InputSchema McpToolsSchema `json:"inputSchema,omitempty"`
}

// Enhanced authorization configuration for tools
type AuthorizationConfig struct {
	// Required Azure roles for this tool
	RequiredRoles []string `yaml:"requiredRoles"`
	// Required database permissions
	RequiredPermissions []string `yaml:"requiredPermissions"`
	// Allowed database operations (SELECT, INSERT, UPDATE, DELETE)
	AllowedOperations []string `yaml:"allowedOperations"`
	// Restricted tables that this tool cannot access
	RestrictedTables []string `yaml:"restrictedTables"`
	// Maximum number of rows that can be affected
	MaxAffectedRows int `yaml:"maxAffectedRows"`
}

// Helper function that returns if a tool invocation request is authorized
func IsAuthorized(authRequiredSources []string, verifiedAuthServices []string) bool {
	if len(authRequiredSources) == 0 {
		// no authorization requirement
		return true
	}
	for _, a := range authRequiredSources {
		if slices.Contains(verifiedAuthServices, a) {
			return true
		}
	}
	return false
}

// IsAuthorizedWithRoles checks if a tool invocation is authorized based on auth services and roles
// This is an enhanced version that supports role-based access control
func IsAuthorizedWithRoles(authRequiredSources []string, verifiedAuthServices []string, claims map[string]map[string]any, requiredRoles []string) bool {
	// First check if basic auth is satisfied
	if !IsAuthorized(authRequiredSources, verifiedAuthServices) {
		return false
	}
	
	// If no roles required, authorization is granted
	if len(requiredRoles) == 0 {
		return true
	}
	
	// Check roles for each verified auth service
	for _, authService := range verifiedAuthServices {
		if claims, exists := claims[authService]; exists {
			// For Azure auth service, we can check roles
			if authService == "azure" {
				// This would need to be implemented with proper dependency injection
				// For now, we'll do a basic check here
				if roles, exists := claims["roles"]; exists {
					var userRoles []string
					switch v := roles.(type) {
					case []string:
						userRoles = v
					case []interface{}:
						for _, role := range v {
							if str, ok := role.(string); ok {
								userRoles = append(userRoles, str)
							}
						}
					}
					
					// Check if user has any of the required roles
					for _, requiredRole := range requiredRoles {
						for _, userRole := range userRoles {
							if userRole == requiredRole {
								return true
							}
						}
					}
				}
			}
		}
	}
	
	return false
}

// Enhanced authorization check with database permission validation
func IsAuthorizedWithDatabasePermissions(
	authRequiredSources []string, 
	verifiedAuthServices []string, 
	claims map[string]map[string]any, 
	authConfig AuthorizationConfig,
	userEmail string,
) bool {
	// 1. Check basic authentication
	if !IsAuthorized(authRequiredSources, verifiedAuthServices) {
		return false
	}
	
	// 2. Check Azure roles
	if len(authConfig.RequiredRoles) > 0 {
		if !IsAuthorizedWithRoles(authRequiredSources, verifiedAuthServices, claims, authConfig.RequiredRoles) {
			return false
		}
	}
	
	// 3. Check database permissions (if configured)
	if len(authConfig.RequiredPermissions) > 0 {
		// This would integrate with your database permission system
		// For now, we'll log the check
		fmt.Printf("[Auth] Checking database permissions for user %s: %v\n", userEmail, authConfig.RequiredPermissions)
	}
	
	return true
}

// DetectPrivilegeEscalation checks for potential privilege escalation attempts
func DetectPrivilegeEscalation(
	userRoles []string,
	requestedOperation string,
	requestedTables []string,
	authConfig AuthorizationConfig,
) (bool, string) {
	
	// Check for restricted table access
	for _, table := range requestedTables {
		for _, restrictedTable := range authConfig.RestrictedTables {
			if table == restrictedTable {
				return true, fmt.Sprintf("Access to restricted table '%s' attempted", table)
			}
		}
	}
	
	// Check for unauthorized operations
	if len(authConfig.AllowedOperations) > 0 {
		operationAllowed := false
		for _, allowedOp := range authConfig.AllowedOperations {
			if requestedOperation == allowedOp {
				operationAllowed = true
				break
			}
		}
		if !operationAllowed {
			return true, fmt.Sprintf("Unauthorized operation '%s' attempted", requestedOperation)
		}
	}
	
	// Check for admin-only operations by non-admin users
	adminOperations := []string{"DROP", "TRUNCATE", "ALTER", "CREATE", "GRANT", "REVOKE"}
	isAdminOperation := false
	for _, adminOp := range adminOperations {
		if requestedOperation == adminOp {
			isAdminOperation = true
			break
		}
	}
	
	if isAdminOperation {
		hasAdminRole := false
		for _, role := range userRoles {
			if role == "mcp.admin" || role == "admin" || role == "superuser" {
				hasAdminRole = true
				break
			}
		}
		if !hasAdminRole {
			return true, fmt.Sprintf("Admin operation '%s' attempted by non-admin user", requestedOperation)
		}
	}
	
	return false, ""
}

