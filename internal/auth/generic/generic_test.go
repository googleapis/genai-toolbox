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
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/golang-jwt/jwt/v5"
)

func generateRSAPrivateKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to create RSA private key: %v", err)
	}
	return key
}

func setupJWKSMockServer(t *testing.T, key *rsa.PrivateKey, keyID string) *httptest.Server {
	t.Helper()

	jwksHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
					KID: keyID,
				},
			}
			jwk, err := jwkset.NewJWKFromKey(key.Public(), options)
			if err != nil {
				t.Fatalf("failed to create JWK: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"keys": []jwkset.JWKMarshal{jwk.Marshal()},
			})
			return
		}

		http.NotFound(w, r)
	})

	return httptest.NewServer(jwksHandler)
}

func generateValidToken(t *testing.T, key *rsa.PrivateKey, keyID string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID
	signedString, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return signedString
}

func TestGetClaimsFromHeader(t *testing.T) {
	privateKey := generateRSAPrivateKey(t)
	keyID := "test-key-id"
	server := setupJWKSMockServer(t, privateKey, keyID)
	defer server.Close()

	cfg := Config{
		Name:                "test-generic-auth",
		Type:                "generic",
		Audience:            "my-audience",
		McpEnabled:          false,
		AuthorizationServer: server.URL,
		ScopesRequired:      []string{"read:files"},
	}

	authService, err := cfg.Initialize()
	if err != nil {
		t.Fatalf("failed to initialize auth service: %v", err)
	}

	genericAuth, ok := authService.(*AuthService)
	if !ok {
		t.Fatalf("expected *AuthService, got %T", authService)
	}

	ctx := context.Background()

	tests := []struct {
		name        string
		setupHeader func() http.Header
		wantError   bool
		errContains string
		validate    func(claims map[string]any)
	}{
		{
			name: "valid token",
			setupHeader: func() http.Header {
				token := generateValidToken(t, privateKey, keyID, jwt.MapClaims{
					"aud":   "my-audience",
					"scope": "read:files write:files",
					"sub":   "test-user",
					"exp":   time.Now().Add(time.Hour).Unix(),
				})
				header := http.Header{}
				header.Set("test-generic-auth_token", token)
				return header
			},
			wantError: false,
			validate: func(claims map[string]any) {
				if sub, ok := claims["sub"].(string); !ok || sub != "test-user" {
					t.Errorf("expected sub=test-user, got %v", claims["sub"])
				}
			},
		},
		{
			name: "no header",
			setupHeader: func() http.Header {
				return http.Header{}
			},
			wantError: false,
			validate: func(claims map[string]any) {
				if claims != nil {
					t.Errorf("expected nil claims on missing header, got %v", claims)
				}
			},
		},
		{
			name: "wrong audience",
			setupHeader: func() http.Header {
				token := generateValidToken(t, privateKey, keyID, jwt.MapClaims{
					"aud":   "wrong-audience",
					"scope": "read:files",
					"exp":   time.Now().Add(time.Hour).Unix(),
				})
				header := http.Header{}
				header.Set("test-generic-auth_token", token)
				return header
			},
			wantError:   true,
			errContains: "audience validation failed",
		},
		{
			name: "expired token",
			setupHeader: func() http.Header {
				token := generateValidToken(t, privateKey, keyID, jwt.MapClaims{
					"aud":   "my-audience",
					"scope": "read:files",
					"exp":   time.Now().Add(-1 * time.Hour).Unix(),
				})
				header := http.Header{}
				header.Set("test-generic-auth_token", token)
				return header
			},
			wantError:   true,
			errContains: "token has invalid claims: token is expired", // Custom JWT err string
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			header := tc.setupHeader()
			claims, err := genericAuth.GetClaimsFromHeader(ctx, header)

			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("expected error containing %q, got: %v", tc.errContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tc.validate != nil {
					tc.validate(claims)
				}
			}
		})
	}
}
