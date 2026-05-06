// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package odata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

// AuthStrategy defines the interface for managing authentication and pre-flight security handshakes.
type AuthStrategy interface {
	// Authorize mutates the outgoing HTTP request to apply the necessary credentials or headers.
	Authorize(ctx context.Context, req *http.Request, client *http.Client) error
	// Evict invalidates any cached tokens or sessions (e.g., on a 403 CSRF failure).
	Evict(ctx context.Context, req *http.Request)
}

// BasicAuthStrategy applies standard HTTP Basic Authentication.
type BasicAuthStrategy struct {
	Username string
	Password string
}

func (s *BasicAuthStrategy) Authorize(ctx context.Context, req *http.Request, client *http.Client) error {
	if req.Header.Get("Authorization") == "" {
		req.SetBasicAuth(s.Username, s.Password)
	}
	return nil
}

func (s *BasicAuthStrategy) Evict(ctx context.Context, req *http.Request) {}

// BearerTokenStrategy applies a static HTTP Bearer token.
type BearerTokenStrategy struct {
	Token string
}

func (s *BearerTokenStrategy) Authorize(ctx context.Context, req *http.Request, client *http.Client) error {
	if req.Header.Get("Authorization") == "" && s.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.Token)
	}
	return nil
}

func (s *BearerTokenStrategy) Evict(ctx context.Context, req *http.Request) {}

// TlsStrategy is a no-op for headers as security is handled at the TLS transport layer.
type TlsStrategy struct{}

func (s *TlsStrategy) Authorize(ctx context.Context, req *http.Request, client *http.Client) error {
	return nil
}

func (s *TlsStrategy) Evict(ctx context.Context, req *http.Request) {}

// DynamicUserOauthStrategy applies the user context OAuth token dynamically from headers.
type DynamicUserOauthStrategy struct {
	AuthTokenHeaderName string
}

func (s *DynamicUserOauthStrategy) Authorize(ctx context.Context, req *http.Request, client *http.Client) error {
	authHeader := req.Header.Get(s.AuthTokenHeaderName)
	if authHeader != "" {
		if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			authHeader = "Bearer " + authHeader
		}
		req.Header.Set("Authorization", authHeader)
		req.Header.Set(s.AuthTokenHeaderName, authHeader)
	}
	return nil
}

func (s *DynamicUserOauthStrategy) Evict(ctx context.Context, req *http.Request) {}

// HashCredentialsWithHeader creates a secure SHA-256 hash of the specific authorization header details for session caching.
func HashCredentialsWithHeader(req *http.Request, headerName string) string {
	auth := req.Header.Get(headerName)
	if auth == "" && headerName != "Authorization" {
		auth = req.Header.Get("Authorization") // Technical fallback
	}
	if auth == "" {
		return "anonymous"
	}
	hash := sha256.Sum256([]byte(auth))
	return hex.EncodeToString(hash[:])
}
