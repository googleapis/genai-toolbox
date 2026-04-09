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
	"github.com/googleapis/mcp-toolbox/internal/auth"
	"github.com/googleapis/mcp-toolbox/internal/util"
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
	httpClient := newSecureHTTPClient()

	// Discover OIDC endpoints
	jwksURL, introspectionURL, err := discoverOIDCConfig(httpClient, cfg.AuthorizationServer)
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC config: %w", err)
	}

	// Create the keyfunc to fetch and cache the JWKS in the background
	kf, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("failed to create keyfunc from JWKS URL %s: %w", jwksURL, err)
	}

	a := &AuthService{
		Config:           cfg,
		kf:               kf,
		client:           httpClient,
		introspectionURL: introspectionURL,
	}
	return a, nil
}

func newSecureHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func discoverOIDCConfig(client *http.Client, AuthorizationServer string) (jwksURI string, introspectionEndpoint string, err error) {
	u, err := url.Parse(AuthorizationServer)
	if err != nil {
		return "", "", fmt.Errorf("invalid auth URL")
	}
	if u.Scheme != "https" {
		log.Printf("WARNING: HTTP instead of HTTPS is being used for AuthorizationServer: %s", AuthorizationServer)
	}

	oidcConfigURL, err := url.JoinPath(AuthorizationServer, ".well-known/openid-configuration")
	if err != nil {
		return "", "", err
	}

	resp, err := client.Get(oidcConfigURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch OIDC config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Limit read size to 1MB to prevent memory exhaustion
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", err
	}

	var config struct {
		JwksUri               string `json:"jwks_uri"`
		IntrospectionEndpoint string `json:"introspection_endpoint"`
	}
	if err := json.Unmarshal(body, &config); err != nil {
		return "", "", err
	}

	if config.JwksUri == "" {
		return "", "", fmt.Errorf("jwks_uri not found in config")
	}

	// Sanitize the resulting JWKS URI before returning it
	parsedJWKS, err := url.Parse(config.JwksUri)
	if err != nil {
		return "", "", fmt.Errorf("invalid jwks_uri detected")
	}
	if parsedJWKS.Scheme != "https" {
		log.Printf("WARNING: HTTP instead of HTTPS is being used for JWKS URI: %s", config.JwksUri)
	}

	return config.JwksUri, config.IntrospectionEndpoint, nil
}

var _ auth.AuthService = AuthService{}

// struct used to store auth service info
type AuthService struct {
	Config
	kf               keyfunc.Keyfunc
	client           *http.Client
	introspectionURL string
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

	tokenStr := headerParts[1]

	if isJWTFormat(tokenStr) {
		return a.validateJwtToken(ctx, tokenStr)
	}
	return a.validateOpaqueToken(ctx, tokenStr)
}

func isJWTFormat(token string) bool {
	return strings.Count(token, ".") == 2
}

// validateJwtToken validates a JWT token locally
func (a AuthService) validateJwtToken(ctx context.Context, tokenStr string) error {
	token, err := jwt.Parse(tokenStr, a.kf.Keyfunc)
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

	scopeClaim, _ := claims["scope"].(string)

	return a.validateClaims(ctx, aud, scopeClaim)
}

// validateOpaqueToken validates an opaque token by calling the introspection endpoint
func (a AuthService) validateOpaqueToken(ctx context.Context, tokenStr string) error {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get logger from context: %w", err)
	}

	introspectionURL := a.introspectionURL
	if introspectionURL == "" {
		introspectionURL, err = url.JoinPath(a.AuthorizationServer, "introspect")
		if err != nil {
			return fmt.Errorf("failed to construct introspection URL: %w", err)
		}
	}

	data := url.Values{}
	data.Set("token", tokenStr)

	req, err := http.NewRequestWithContext(ctx, "POST", introspectionURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create introspection request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// Send request to auth server's introspection endpoint
	resp, err := a.client.Do(req)
	if err != nil {
		logger.ErrorContext(ctx, "failed to call introspection endpoint: %v", err)
		return &MCPAuthError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("failed to call introspection endpoint: %v", err), ScopesRequired: a.ScopesRequired}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.WarnContext(ctx, "introspection failed with status: %d", resp.StatusCode)
		return &MCPAuthError{Code: http.StatusUnauthorized, Message: fmt.Sprintf("introspection failed with status: %d", resp.StatusCode), ScopesRequired: a.ScopesRequired}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("failed to read introspection response: %w", err)
	}

	var introspectResp struct {
		Active bool            `json:"active"`
		Scope  string          `json:"scope"`
		Aud    json.RawMessage `json:"aud"`
		Exp    int64           `json:"exp"`
	}

	if err := json.Unmarshal(body, &introspectResp); err != nil {
		return fmt.Errorf("failed to parse introspection response: %w", err)
	}

	if !introspectResp.Active {
		logger.InfoContext(ctx, "token is not active")
		return &MCPAuthError{Code: http.StatusUnauthorized, Message: "token is not active", ScopesRequired: a.ScopesRequired}
	}

	// Verify expiration (with 1 minute leeway)
	const leeway = 60
	if introspectResp.Exp > 0 && time.Now().Unix() > (introspectResp.Exp+leeway) {
		logger.WarnContext(ctx, "token has expired: exp=%d, now=%d", introspectResp.Exp, time.Now().Unix())
		return &MCPAuthError{Code: http.StatusUnauthorized, Message: "token has expired", ScopesRequired: a.ScopesRequired}
	}

	// Extract audience
	// According to RFC 7662, the aud claim can be a string or an array of strings
	var aud []string
	if len(introspectResp.Aud) > 0 {
		var audStr string
		var audArr []string
		if err := json.Unmarshal(introspectResp.Aud, &audStr); err == nil {
			aud = []string{audStr}
		} else if err := json.Unmarshal(introspectResp.Aud, &audArr); err == nil {
			aud = audArr
		} else {
			logger.WarnContext(ctx, "failed to parse aud claim in introspection response")
			return &MCPAuthError{Code: http.StatusUnauthorized, Message: "invalid aud claim", ScopesRequired: a.ScopesRequired}
		}
	}

	return a.validateClaims(ctx, aud, introspectResp.Scope)
}

// validateClaims validates the audience and scopes of a token
func (a AuthService) validateClaims(ctx context.Context, aud []string, scopeStr string) error {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get logger from context: %w", err)
	}

	// Validate audience
	if a.Audience != "" {
		isAudValid := false
		for _, audItem := range aud {
			if audItem == a.Audience {
				isAudValid = true
				break
			}
		}

		if !isAudValid {
			logger.WarnContext(ctx, "audience validation failed: expected %s", a.Audience)
			return &MCPAuthError{Code: http.StatusUnauthorized, Message: "audience validation failed", ScopesRequired: a.ScopesRequired}
		}
	}

	// Check scopes
	if len(a.ScopesRequired) > 0 {
		tokenScopes := strings.Split(scopeStr, " ")
		scopeMap := make(map[string]bool)
		for _, s := range tokenScopes {
			scopeMap[s] = true
		}

		for _, requiredScope := range a.ScopesRequired {
			if !scopeMap[requiredScope] {
				logger.WarnContext(ctx, "insufficient scopes: missing %s", requiredScope)
				return &MCPAuthError{Code: http.StatusForbidden, Message: "insufficient scopes", ScopesRequired: a.ScopesRequired}
			}
		}
	}

	return nil
}
