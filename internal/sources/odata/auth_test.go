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

package odata

import (
	"context"
	"net/http"
	"testing"
)

func TestHashCredentialsWithHeader(t *testing.T) {
	req, err := http.NewRequest("GET", "https://example.com", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	// 1. Missing auth headers -> "anonymous"
	if got := HashCredentialsWithHeader(req, "X-Custom-Auth"); got != "anonymous" {
		t.Errorf("expected 'anonymous' for missing header, got %q", got)
	}

	// 2. Technical fallback to Authorization header
	req.Header.Set("Authorization", "Bearer token-value-123")
	hash1 := HashCredentialsWithHeader(req, "X-Custom-Auth")
	if hash1 == "anonymous" || hash1 == "" {
		t.Errorf("expected valid sha256 hash for Authorization fallback, got %q", hash1)
	}

	// 3. Exact custom header match
	req.Header.Set("X-Custom-Auth", "Bearer custom-token-456")
	hash2 := HashCredentialsWithHeader(req, "X-Custom-Auth")
	if hash2 == hash1 {
		t.Errorf("expected different hash when X-Custom-Auth is present")
	}
}

func TestBasicAuthStrategy_Authorize(t *testing.T) {
	strategy := &BasicAuthStrategy{Username: "user1", Password: "pass1"}
	req, _ := http.NewRequest("GET", "https://example.com", nil)

	if err := strategy.Authorize(context.Background(), req, nil); err != nil {
		t.Fatalf("Authorize failed: %v", err)
	}

	user, pass, ok := req.BasicAuth()
	if !ok || user != "user1" || pass != "pass1" {
		t.Errorf("unexpected basic auth: user=%q, pass=%q, ok=%v", user, pass, ok)
	}
}

func TestBearerTokenStrategy_Authorize(t *testing.T) {
	strategy := &BearerTokenStrategy{Token: "my-secret-token"}
	req, _ := http.NewRequest("GET", "https://example.com", nil)

	if err := strategy.Authorize(context.Background(), req, nil); err != nil {
		t.Fatalf("Authorize failed: %v", err)
	}

	if got := req.Header.Get("Authorization"); got != "Bearer my-secret-token" {
		t.Errorf("unexpected Authorization header: %q", got)
	}
}

func TestDynamicUserOauthStrategy_Authorize(t *testing.T) {
	strategy := &DynamicUserOauthStrategy{AuthTokenHeaderName: "X-SAP-Auth"}
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	req.Header.Set("X-SAP-Auth", "user-dynamic-token")

	if err := strategy.Authorize(context.Background(), req, nil); err != nil {
		t.Fatalf("Authorize failed: %v", err)
	}

	if got := req.Header.Get("Authorization"); got != "Bearer user-dynamic-token" {
		t.Errorf("unexpected Authorization header: %q", got)
	}
	if got := req.Header.Get("X-SAP-Auth"); got != "Bearer user-dynamic-token" {
		t.Errorf("unexpected X-SAP-Auth header: %q", got)
	}
}
