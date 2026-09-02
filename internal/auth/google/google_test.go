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

package google_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/auth"
	"github.com/googleapis/mcp-toolbox/internal/auth/google"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
)

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

			a := google.NewAuthServiceForTest(google.Config{
				Audience: tc.audience,
				ClientID: tc.clientID,
			}, mockClient)

			header := make(http.Header)
			header.Set("Authorization", "Bearer some-opaque-token")

			_, err := a.ValidateMCPAuth(context.Background(), header)
			if (err != nil) != tc.wantError {
				t.Fatalf("ValidateMCPAuth() returned error: %v, wantError: %v", err, tc.wantError)
			}
		})
	}
}

func TestParseFromYaml(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.AuthServiceConfigs
	}{
		{
			desc: "only clientId, mcpEnabled false",
			in: `
			kind: authService
			name: my-google-auth
			type: google
			clientId: my-client-id
			`,
			want: map[string]auth.AuthServiceConfig{
				"my-google-auth": google.Config{
					Name:       "my-google-auth",
					Type:       google.AuthServiceType,
					ClientID:   "my-client-id",
					McpEnabled: false,
				},
			},
		},
		{
			desc: "only audience, mcpEnabled true",
			in: `
			kind: authService
			name: my-google-auth
			type: google
			audience: my-audience
			mcpEnabled: true
			`,
			want: map[string]auth.AuthServiceConfig{
				"my-google-auth": google.Config{
					Name:       "my-google-auth",
					Type:       google.AuthServiceType,
					Audience:   "my-audience",
					McpEnabled: true,
				},
			},
		},
		{
			desc: "scopesRequired, mcpEnabled true",
			in: `
			kind: authService
			name: my-google-auth
			type: google
			scopesRequired:
			  - email
			mcpEnabled: true
			`,
			want: map[string]auth.AuthServiceConfig{
				"my-google-auth": google.Config{
					Name:           "my-google-auth",
					Type:           google.AuthServiceType,
					ScopesRequired: []string{"email"},
					McpEnabled:     true,
				},
			},
		},
		{
			desc: "both clientId and audience, mcpEnabled true",
			in: `
			kind: authService
			name: my-google-auth
			type: google
			clientId: my-client-id
			audience: my-audience
			mcpEnabled: true
			`,
			want: map[string]auth.AuthServiceConfig{
				"my-google-auth": google.Config{
					Name:       "my-google-auth",
					Type:       google.AuthServiceType,
					ClientID:   "my-client-id",
					Audience:   "my-audience",
					McpEnabled: true,
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, got, _, _, _, _, err := server.UnmarshalPrimitiveConfig(context.Background(), testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if !cmp.Equal(tc.want, got) {
				t.Fatalf("incorrect parse: want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestFailParseFromYaml(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "only audience, mcpEnabled false",
			in: `
			kind: authService
			name: my-google-auth
			type: google
			audience: my-audience
			`,
			err: "error unmarshaling authService: `audience` is not allowed when `mcpEnabled` is false",
		},
		{
			desc: "scopesRequired, mcpEnabled false",
			in: `
			kind: authService
			name: my-google-auth
			type: google
			scopesRequired:
			  - email
			`,
			err: "error unmarshaling authService: `scopesRequired` is not allowed when `mcpEnabled` is false",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, _, _, _, err := server.UnmarshalPrimitiveConfig(context.Background(), testutils.FormatYaml(tc.in))
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			errStr := err.Error()
			if !strings.Contains(errStr, tc.err) {
				t.Fatalf("unexpected error: got %q, want it to contain %q", errStr, tc.err)
			}
		})
	}
}
