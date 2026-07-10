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

package google

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"google.golang.org/api/idtoken"
)

func TestInitialize_Validation(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name: "only clientID, mcpEnabled false",
			config: Config{
				Name:       "google-auth",
				Type:       "google",
				ClientID:   "my-client-id",
				McpEnabled: false,
			},
			wantError: false,
		},
		{
			name: "only audience, mcpEnabled false (disallowed)",
			config: Config{
				Name:       "google-auth",
				Type:       "google",
				Audience:   "my-audience",
				McpEnabled: false,
			},
			wantError: true,
		},
		{
			name: "only audience, mcpEnabled true (allowed)",
			config: Config{
				Name:       "google-auth",
				Type:       "google",
				Audience:   "my-audience",
				McpEnabled: true,
			},
			wantError: false,
		},
		{
			name: "scopesRequired, mcpEnabled false (disallowed)",
			config: Config{
				Name:           "google-auth",
				Type:           "google",
				ScopesRequired: []string{"scope"},
				McpEnabled:     false,
			},
			wantError: true,
		},
		{
			name: "scopesRequired, mcpEnabled true, with audience (allowed)",
			config: Config{
				Name:           "google-auth",
				Type:           "google",
				ScopesRequired: []string{"scope"},
				Audience:       "my-audience",
				McpEnabled:     true,
			},
			wantError: false,
		},
		{
			name: "scopesRequired, mcpEnabled true, without audience or clientID (disallowed)",
			config: Config{
				Name:           "google-auth",
				Type:           "google",
				ScopesRequired: []string{"scope"},
				McpEnabled:     true,
			},
			wantError: true,
		},
		{
			name: "both clientID and audience, mcpEnabled true",
			config: Config{
				Name:       "google-auth",
				Type:       "google",
				ClientID:   "my-client-id",
				Audience:   "my-audience",
				McpEnabled: true,
			},
			wantError: false,
		},
		{
			// Without clientId in non-MCP mode, GetClaimsFromHeader would
			// validate ID tokens with an empty audience, which makes
			// idtoken.Validate skip the audience check and accept any
			// Google-signed token. Initialize must now reject this config.
			name: "neither clientID nor audience, mcpEnabled false (disallowed)",
			config: Config{
				Name:       "google-auth",
				Type:       "google",
				McpEnabled: false,
			},
			wantError: true,
		},
		{
			name: "neither clientID nor audience, mcpEnabled true (disallowed)",
			config: Config{
				Name:       "google-auth",
				Type:       "google",
				McpEnabled: true,
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.config.Initialize()
			if (err != nil) != tc.wantError {
				t.Fatalf("Initialize() returned error: %v, wantError: %v", err, tc.wantError)
			}
		})
	}
}

func TestGetClaimsFromHeader_NoToken(t *testing.T) {
	a := AuthService{Config: Config{Name: "google-auth", ClientID: "my-client-id"}}

	claims, err := a.GetClaimsFromHeader(context.Background(), make(http.Header))
	if err != nil {
		t.Fatalf("GetClaimsFromHeader() with no token returned error: %v", err)
	}
	if claims != nil {
		t.Fatalf("GetClaimsFromHeader() with no token returned claims: %v, want nil", claims)
	}
}

func TestGetClaimsFromHeader_AudienceBinding(t *testing.T) {
	const tokenAud = "111-token.apps.googleusercontent.com"

	// Substitute a fake validator that models idtoken.Validate's audience
	// check (signature verification is assumed to pass). The real library
	// performs `if audience != "" && payload.Audience != audience`, so an
	// empty audience means no audience check is enforced.
	orig := validateIDToken
	t.Cleanup(func() { validateIDToken = orig })
	validateIDToken = func(_ context.Context, _ string, audience string) (*idtoken.Payload, error) {
		if audience != "" && audience != tokenAud {
			return nil, fmt.Errorf("idtoken: audience provided does not match aud claim in the JWT")
		}
		return &idtoken.Payload{
			Audience: tokenAud,
			Claims:   map[string]any{"aud": tokenAud, "email": "user@example.com"},
		}, nil
	}

	tests := []struct {
		name      string
		clientID  string
		wantError bool
	}{
		{
			name:      "configured clientId matches token aud",
			clientID:  tokenAud,
			wantError: false,
		},
		{
			name:      "configured clientId does not match token aud",
			clientID:  "999-other.apps.googleusercontent.com",
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := AuthService{Config: Config{Name: "google-auth", ClientID: tc.clientID}}
			header := make(http.Header)
			header.Set("google-auth_token", "eyJ.fake.token")

			claims, err := a.GetClaimsFromHeader(context.Background(), header)
			if (err != nil) != tc.wantError {
				t.Fatalf("GetClaimsFromHeader() returned error: %v, wantError: %v", err, tc.wantError)
			}
			if !tc.wantError && claims["aud"] != tokenAud {
				t.Fatalf("GetClaimsFromHeader() aud claim = %v, want %q", claims["aud"], tokenAud)
			}
		})
	}
}

type mockRoundTripper func(req *http.Request) (*http.Response, error)

func (f mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestValidateMCPAuth_Opaque_Fallback(t *testing.T) {
	tests := []struct {
		name         string
		audience     string
		clientID     string
		tokenInfoAud string
		tokenInfoAzp string
		wantError    bool
	}{
		{
			name:         "only audience matches",
			audience:     "my-aud",
			tokenInfoAud: "my-aud",
			wantError:    false,
		},
		{
			name:         "only clientID matches (fallback)",
			clientID:     "my-client-id",
			tokenInfoAud: "my-client-id",
			wantError:    false,
		},
		{
			name:         "only clientID, tokenInfo uses azp (fallback)",
			clientID:     "my-client-id",
			tokenInfoAzp: "my-client-id",
			wantError:    false,
		},
		{
			name:         "both audience and clientID, audience matches",
			audience:     "my-aud",
			clientID:     "my-client-id",
			tokenInfoAud: "my-aud",
			wantError:    false,
		},
		{
			name:         "both audience and clientID, clientID does not fall back if audience is specified",
			audience:     "my-aud",
			clientID:     "my-client-id",
			tokenInfoAud: "my-client-id",
			wantError:    true,
		},
		{
			name:         "neither audience nor clientID specified",
			tokenInfoAud: "any-aud",
			wantError:    false,
		},
		{
			name:         "audience mismatch",
			audience:     "my-aud",
			tokenInfoAud: "wrong-aud",
			wantError:    true,
		},
		{
			name:         "clientID mismatch (fallback)",
			clientID:     "my-client-id",
			tokenInfoAud: "wrong-aud",
			wantError:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &http.Client{
				Transport: mockRoundTripper(func(req *http.Request) (*http.Response, error) {
					if req.URL.String() != "https://oauth2.googleapis.com/tokeninfo" {
						return nil, fmt.Errorf("unexpected URL: %s", req.URL.String())
					}
					respBody := fmt.Sprintf(`{"aud": %q, "azp": %q, "scope": "openid email"}`, tc.tokenInfoAud, tc.tokenInfoAzp)
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(respBody)),
						Header:     make(http.Header),
					}, nil
				}),
			}

			a := AuthService{
				Config: Config{
					Audience: tc.audience,
					ClientID: tc.clientID,
				},
				client: mockClient,
			}

			header := make(http.Header)
			header.Set("Authorization", "Bearer some-opaque-token")

			_, err := a.ValidateMCPAuth(context.Background(), header)
			if (err != nil) != tc.wantError {
				t.Fatalf("ValidateMCPAuth() returned error: %v, wantError: %v", err, tc.wantError)
			}
		})
	}
}
