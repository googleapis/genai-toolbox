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
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionLRUCache(t *testing.T) {
	cache := newSessionCache(2, 10*time.Minute)

	if got := cache.Get("nonexistent"); got != nil {
		t.Errorf("expected nil for non-existent key, got %v", got)
	}

	session1 := &UserSession{CsrfToken: "token1", ExpiresAt: time.Now().Add(10 * time.Minute)}
	cache.Set("key1", session1)

	if got := cache.Get("key1"); got == nil || got.CsrfToken != "token1" {
		t.Errorf("failed to retrieve key1 from cache")
	}

	session2 := &UserSession{CsrfToken: "token2", ExpiresAt: time.Now().Add(10 * time.Minute)}
	session3 := &UserSession{CsrfToken: "token3", ExpiresAt: time.Now().Add(10 * time.Minute)}

	cache.Set("key2", session2)
	cache.Set("key3", session3) // Evicts key1 since maxItems=2

	if got := cache.Get("key1"); got != nil {
		t.Errorf("expected key1 to be evicted when maxItems exceeded")
	}

	cache.Remove("key2")
	if got := cache.Get("key2"); got != nil {
		t.Errorf("expected key2 to be removed")
	}
}

func TestGatewayStrategy_AuthorizeAndEvict(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-CSRF-Token", "mock-csrf-token")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	primary := &BearerTokenStrategy{Token: "test-token"}
	strategy := NewGatewayStrategy(ts.URL, nil, primary, nil, "Authorization")

	req, _ := http.NewRequest("POST", ts.URL+"/FlightCollection", nil)
	client := ts.Client()

	if err := strategy.Authorize(context.Background(), req, client); err != nil {
		t.Fatalf("Authorize failed: %v", err)
	}

	if got := req.Header.Get("X-CSRF-Token"); got != "mock-csrf-token" {
		t.Errorf("expected X-CSRF-Token 'mock-csrf-token', got %q", got)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer test-token" {
		t.Errorf("expected Authorization header 'Bearer test-token', got %q", got)
	}

	// Verify caching and Evict
	jar, _ := cookiejar.New(nil)
	cachedSession := strategy.sessionCache.Get(HashCredentialsWithHeader(req, "Authorization"))
	if cachedSession == nil {
		t.Fatalf("expected session to be cached after pre-flight")
	}
	cachedSession.Jar = jar

	strategy.Evict(context.Background(), req)
	if got := strategy.sessionCache.Get(HashCredentialsWithHeader(req, "Authorization")); got != nil {
		t.Errorf("expected session to be cleared on Evict")
	}
}
