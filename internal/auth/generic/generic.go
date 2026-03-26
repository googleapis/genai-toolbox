// Copyright 2026 Google LLC
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

package generic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/googleapis/genai-toolbox/internal/auth"
)

const AuthServiceType string = "generic"

// validate interface
var _ auth.AuthServiceConfig = Config{}

// Auth service configuration
type Config struct {
	Name                string   `yaml:"name" validate:"required"`
	Type                string   `yaml:"type" validate:"required"`
	Audience            string   `yaml:"audience" validate:"required"`
	McpEnabled          bool     `yaml:"mcpEnabled"`
	AuthorizationServer string   `yaml:"authorizationServer" validate:"required"`
	ScopesRequired      []string `yaml:"scopesRequired"`
}

// Returns the auth service type
func (cfg Config) AuthServiceConfigType() string {
	return AuthServiceType
}

// Initialize a generic auth service
func (cfg Config) Initialize() (auth.AuthService, error) {
	// Discover the JWKS URL from the OIDC configuration endpoint
	jwksURL, err := discoverJWKSURL(cfg.AuthorizationServer)
	if err != nil {
		return nil, fmt.Errorf("failed to discover JWKS URL: %w", err)
	}

	// Create the keyfunc to fetch and cache the JWKS in the background
	kf, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("failed to create keyfunc from JWKS URL %s: %w", jwksURL, err)
	}

	a := &AuthService{
		Config: cfg,
		kf:     kf,
	}
	return a, nil
}

func discoverJWKSURL(AuthorizationServer string) (string, error) {
	u, err := url.Parse(AuthorizationServer)
	if err != nil {
		return "", fmt.Errorf("invalid auth URL")
	}
	if u.Scheme != "https" {
		log.Printf("WARNING: HTTP instead of HTTPS is being used for AuthorizationServer: %s", AuthorizationServer)
	}

	oidcConfigURL, err := url.JoinPath(AuthorizationServer, ".well-known/openid-configuration")
	if err != nil {
		return "", err
	}

	// HTTP Client
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		// Prevent redirect loops or redirects to internal sites
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(oidcConfigURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch OIDC config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Limit read size to 1MB to prevent memory exhaustion
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}

	var config struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.Unmarshal(body, &config); err != nil {
		return "", err
	}

	if config.JWKSURI == "" {
		return "", fmt.Errorf("jwks_uri not found in config")
	}

	// Sanitize the resulting JWKS URI before returning it
	parsedJWKS, err := url.Parse(config.JWKSURI)
	if err != nil {
		return "", fmt.Errorf("invalid jwks_uri detected")
	}
	if parsedJWKS.Scheme != "https" {
		log.Printf("WARNING: HTTP instead of HTTPS is being used for JWKS URI: %s", config.JWKSURI)
	}

	return config.JWKSURI, nil
}

var _ auth.AuthService = AuthService{}

// struct used to store auth service info
type AuthService struct {
	Config
	kf keyfunc.Keyfunc
}

// Returns the auth service type
func (a AuthService) AuthServiceType() string {
	return AuthServiceType
}

func (a AuthService) ToConfig() auth.AuthServiceConfig {
	return a.Config
}

// Returns the name of the auth service
func (a AuthService) GetName() string {
	return a.Name
}

// Verifies generic JWT access token inside the Authorization header
func (a AuthService) GetClaimsFromHeader(ctx context.Context, h http.Header) (map[string]any, error) {
	if a.McpEnabled {
		return nil, nil
	}

	tokenString := h.Get(a.Name + "_token")
	if tokenString == "" {
		return nil, nil
	}

	// Parse and verify the token signature
	token, err := jwt.Parse(tokenString, a.kf.Keyfunc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse and verify JWT token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid JWT token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid JWT claims format")
	}

	// Validate 'aud' (audience) claim
	aud, err := claims.GetAudience()
	if err != nil {
		return nil, fmt.Errorf("could not parse audience from token: %w", err)
	}

	isAudValid := false
	for _, audItem := range aud {
		if audItem == a.Audience {
			isAudValid = true
			break
		}
	}

	if !isAudValid {
		return nil, fmt.Errorf("audience validation failed: expected %s, got %v", a.Audience, aud)
	}

	return claims, nil
}

// MCPAuthError represents an error during MCP authentication validation.
type MCPAuthError struct {
	Code           int
	Message        string
	ScopesRequired []string
}

func (e *MCPAuthError) Error() string { return e.Message }

// ValidateMCPAuth handles MCP auth token validation
func (a AuthService) ValidateMCPAuth(ctx context.Context, h http.Header) error {
	tokenString := h.Get("Authorization")
	if tokenString == "" {
		return &MCPAuthError{Code: http.StatusUnauthorized, Message: "missing access token", ScopesRequired: a.ScopesRequired}
	}

	headerParts := strings.Split(tokenString, " ")
	if len(headerParts) != 2 || strings.ToLower(headerParts[0]) != "bearer" {
		return &MCPAuthError{Code: http.StatusUnauthorized, Message: "authorization header must be in the format 'Bearer <token>'", ScopesRequired: a.ScopesRequired}
	}

	token, err := jwt.Parse(headerParts[1], a.kf.Keyfunc)
	if err != nil || !token.Valid {
		return &MCPAuthError{Code: http.StatusUnauthorized, Message: "invalid or expired token", ScopesRequired: a.ScopesRequired}
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return &MCPAuthError{Code: http.StatusUnauthorized, Message: "invalid JWT claims format", ScopesRequired: a.ScopesRequired}
	}

	// Validate audience
	aud, err := claims.GetAudience()
	if err != nil {
		return &MCPAuthError{Code: http.StatusUnauthorized, Message: "could not parse audience from token", ScopesRequired: a.ScopesRequired}
	}

	isAudValid := false
	for _, audItem := range aud {
		if audItem == a.Audience {
			isAudValid = true
			break
		}
	}

	if !isAudValid {
		return &MCPAuthError{Code: http.StatusUnauthorized, Message: "audience validation failed", ScopesRequired: a.ScopesRequired}
	}

	// Check scopes
	if len(a.ScopesRequired) > 0 {
		scopeClaim, ok := claims["scope"].(string)
		if !ok {
			return &MCPAuthError{Code: http.StatusForbidden, Message: "insufficient scopes", ScopesRequired: a.ScopesRequired}
		}

		tokenScopes := strings.Split(scopeClaim, " ")
		scopeMap := make(map[string]bool)
		for _, s := range tokenScopes {
			scopeMap[s] = true
		}

		for _, requiredScope := range a.ScopesRequired {
			if !scopeMap[requiredScope] {
				return &MCPAuthError{Code: http.StatusForbidden, Message: "insufficient scopes", ScopesRequired: a.ScopesRequired}
			}
		}
	}

	return nil
}
