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

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/golang-jwt/jwt/v5"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"
)

// TestMcpAuth test for MCP Authorization
func TestMcpAuth(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// Set up generic auth mock server
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to create RSA private key: %v", err)
	}
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"issuer":   "https://example.com",
				"jwks_uri": "http://" + r.Host + "/jwks",
			})
			return
		}
		if r.URL.Path == "/jwks" {
			options := jwkset.JWKOptions{
				Metadata: jwkset.JWKMetadataOptions{
					KID: "test-key-id",
				},
			}
			jwk, _ := jwkset.NewJWKFromKey(privateKey.Public(), options)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"keys": []jwkset.JWKMarshal{jwk.Marshal()},
			})
			return
		}
		if r.URL.Path == "/introspect" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"active": true,
				"scope":  "read:files",
				"aud":    "test-audience",
				"exp":    time.Now().Add(time.Hour).Unix(),
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer jwksServer.Close()

	toolsFile := map[string]any{
		"sources": map[string]any{},
		"authServices": map[string]any{
			"my-generic-auth": map[string]any{
				"type":                "generic",
				"audience":            "test-audience",
				"authorizationServer": jwksServer.URL,
				"scopesRequired":      []string{"read:files"},
				"mcpEnabled":          true,
			},
		},
		"tools": map[string]any{},
	}
	args := []string{"--enable-api", "--toolbox-url=http://127.0.0.1:5000"}
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	api := "http://127.0.0.1:5000/mcp/sse"

	// Generate invalid token (wrong scopes)
	invalidToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"aud":   "test-audience",
		"scope": "wrong:scope",
		"sub":   "test-user",
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	invalidToken.Header["kid"] = "test-key-id"
	invalidSignedString, _ := invalidToken.SignedString(privateKey)

	// Generate valid token (correct scopes)
	validToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"aud":   "test-audience",
		"scope": "read:files",
		"sub":   "test-user",
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	validToken.Header["kid"] = "test-key-id"
	validSignedString, _ := validToken.SignedString(privateKey)

	tests := []struct {
		name           string
		token          string
		wantStatusCode int
		checkWWWAuth   func(t *testing.T, authHeader string)
	}{
		{
			name:           "401 Unauthorized without token",
			token:          "",
			wantStatusCode: http.StatusUnauthorized,
			checkWWWAuth: func(t *testing.T, authHeader string) {
				if !strings.Contains(authHeader, `resource_metadata="http://127.0.0.1:5000/.well-known/oauth-protected-resource"`) || !strings.Contains(authHeader, `scope="read:files"`) {
					t.Fatalf("expected WWW-Authenticate header to contain resource_metadata and scope, got: %s", authHeader)
				}
			},
		},
		{
			name:           "403 Forbidden with insufficient scopes",
			token:          invalidSignedString,
			wantStatusCode: http.StatusForbidden,
			checkWWWAuth: func(t *testing.T, authHeader string) {
				if !strings.Contains(authHeader, `resource_metadata="http://127.0.0.1:5000/.well-known/oauth-protected-resource"`) || !strings.Contains(authHeader, `scope="read:files"`) || !strings.Contains(authHeader, `error="insufficient_scope"`) {
					t.Fatalf("expected WWW-Authenticate header to contain error, scope, and resource_metadata, got: %s", authHeader)
				}
			},
		},
		{
			name:           "200 OK with valid token",
			token:          validSignedString,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "200 OK with valid opaque token",
			token:          "this-is-an-opaque-token",
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, api, nil)
			if tc.token != "" {
				req.Header.Add("Authorization", "Bearer "+tc.token)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatusCode {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected %d, got %d: %s", tc.wantStatusCode, resp.StatusCode, string(bodyBytes))
			}

			if tc.checkWWWAuth != nil {
				authHeader := resp.Header.Get("WWW-Authenticate")
				tc.checkWWWAuth(t, authHeader)
			}
		})
	}
}
