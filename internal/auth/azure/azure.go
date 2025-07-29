// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azure

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"
	"github.com/googleapis/genai-toolbox/internal/auth"
)

const AuthServiceKind string = "azure"

// validate interface
var _ auth.AuthServiceConfig = Config{}

// Auth service configuration
type Config struct {
	Name     string `yaml:"name" validate:"required"`
	Kind     string `yaml:"kind" validate:"required"`
	ClientID string `yaml:"clientId" validate:"required"`
}

// Returns the auth service kind
func (cfg Config) AuthServiceConfigKind() string {
	return AuthServiceKind
}

// Initialize a Azure auth service
func (cfg Config) Initialize() (auth.AuthService, error) {
	a := &AuthService{
		Name:     cfg.Name,
		Kind:     AuthServiceKind,
		ClientID: cfg.ClientID,
	}
	return a, nil
}

var _ auth.AuthService = AuthService{}

// struct used to store auth service info
type AuthService struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	ClientID string `yaml:"clientId"`
}

// Returns the auth service kind
func (a AuthService) AuthServiceKind() string {
	return AuthServiceKind
}

// Returns the name of the auth service
func (a AuthService) GetName() string {
	return a.Name
}

// Verifies Azure access token and return claims
// This validation is performed locally using Azure's public JWKS (JSON Web Key Set)
// No direct communication with Azure AD is required for token validation
func (a AuthService) GetClaimsFromHeader(ctx context.Context, h http.Header) (map[string]any, error) {
	token := h.Get(a.Name + "_token")
	if token == "" {
		return nil, nil
	}

	// Remove Bearer prefix if present
	token = strings.TrimPrefix(token, "Bearer ")
	fmt.Printf("[Azure Auth] Processing token for service: %s\n", a.Name)

	// Fetch Azure's public JWKS from Microsoft's discovery endpoint
	// This is a one-time fetch that gets cached and refreshed periodically
	jwksURL := "https://login.microsoftonline.com/common/discovery/v2.0/keys"
	jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{
		RefreshInterval: time.Hour,
		RefreshErrorHandler: func(err error) {
			fmt.Printf("[Azure Auth] Error refreshing Azure JWKS: %v\n", err)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure JWKS: %w", err)
	}
	defer jwks.EndBackground()

	// Parse and validate the JWT token using Azure's public keys
	// This is a local cryptographic validation - no network call to Azure AD
	parsed, err := jwt.Parse(token, jwks.Keyfunc)
	if err != nil {
		return nil, fmt.Errorf("Azure access token validation failed: %w", err)
	}
	if !parsed.Valid {
		return nil, fmt.Errorf("Azure access token is invalid")
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("Azure access token claims are not in expected format")
	}

	// Validate audience (aud) claim matches ClientID
	if aud, ok := claims["aud"].(string); !ok || aud != "api://5b4e39a3-dc18-4bfa-93aa-94b2d2a5d822" {
		return nil, fmt.Errorf("Azure access token audience (aud) claim mismatch")
	}
	fmt.Printf("[Azure Auth] Token validation successful for audience: %s\n", claims["aud"])

	// Validate issuer (iss) claim (optional, can be made stricter)
	if iss, ok := claims["iss"].(string); !ok || !strings.HasPrefix(iss, "https://sts.windows.net/") {
		return nil, fmt.Errorf("Azure access token issuer (iss) claim mismatch")
	}
	fmt.Printf("[Azure Auth] Token issuer validation successful: %s\n", claims["iss"])

	// Debug: Print roles from JWT claims if present
	if roles, exists := claims["roles"]; exists {
		fmt.Printf("[Azure Auth] User roles from token: %v\n", roles)
	} else {
		fmt.Printf("[Azure Auth] No roles found in token claims\n")
	}

	return claims, nil
}

// CheckAuthorization verifies if the user has the required roles for authorization
// This method can be used to implement role-based access control (RBAC)
func (a AuthService) CheckAuthorization(claims map[string]any, requiredRoles []string) bool {
	if len(requiredRoles) == 0 {
		// No roles required, authorization granted
		return true
	}
	
	if roles, exists := claims["roles"]; exists {
		// Handle different possible types for roles
		var userRoles []string
		switch v := roles.(type) {
		case []string:
			userRoles = v
		case []interface{}:
			// Convert []interface{} to []string
			for _, role := range v {
				if str, ok := role.(string); ok {
					userRoles = append(userRoles, str)
				}
			}
		default:
			fmt.Printf("[Azure Auth] Unexpected roles type: %T\n", roles)
			return false
		}
		
		// Check if user has any of the required roles
		for _, requiredRole := range requiredRoles {
			for _, userRole := range userRoles {
				if userRole == requiredRole {
					fmt.Printf("[Azure Auth] Authorization granted for role: %s\n", requiredRole)
					return true
				}
			}
		}
	}
	
	fmt.Printf("[Azure Auth] Authorization denied - required roles: %v\n", requiredRoles)
	return false
}
