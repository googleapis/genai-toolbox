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

package http

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/golang-jwt/jwt/v5"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

func getMCPHTTPSourceConfig(t *testing.T) map[string]any {
	idToken, err := tests.GetGoogleIdToken(t)
	if err != nil {
		t.Logf("Warning: error getting ID token: %s. Using dummy token.", err)
		idToken = "dummy-token"
	}
	idToken = "Bearer " + idToken

	return map[string]any{
		"type":    HttpSourceType,
		"headers": map[string]string{"Authorization": idToken},
	}
}

func TestHTTPListTools(t *testing.T) {
	// Start a test server with multiTool handler
	server := httptest.NewServer(http.HandlerFunc(multiTool))
	defer server.Close()

	sourceConfig := getMCPHTTPSourceConfig(t)
	sourceConfig["baseUrl"] = server.URL
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// Set up generic auth mock server (copied from legacy test)
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
		http.NotFound(w, r)
	}))
	defer jwksServer.Close()

	toolsFile := getHTTPToolsConfig(sourceConfig, HttpToolType, jwksServer.URL)

	// Start the toolbox server.
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	// Wait for server ready
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	expectedTools := []tests.MCPToolManifest{
		{
			Name:        "my-simple-tool",
			Description: "Simple tool to test end to end functionality.",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}, "required": []any{}},
		},
		{
			Name:        "my-tool",
			Description: "some description",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []any{"id", "name"},
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "integer",
						"description": "user ID",
					},
					"name": map[string]any{
						"type":        "string",
						"description": "user name",
					},
				},
			},
		},
	}

	tests.RunMCPToolsListMethod(t, expectedTools)
}

func TestHTTPCallTool(t *testing.T) {
	// Start a test server with multiTool handler
	server := httptest.NewServer(http.HandlerFunc(multiTool))
	defer server.Close()

	sourceConfig := getMCPHTTPSourceConfig(t)
	sourceConfig["baseUrl"] = server.URL
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// Set up generic auth mock server (needed for config generation)
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
		http.NotFound(w, r)
	}))
	defer jwksServer.Close()

	toolsFile := getHTTPToolsConfig(sourceConfig, HttpToolType, jwksServer.URL)

	// Start the toolbox server.
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	// Wait for server ready
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	// Run Generic Auth Tests
	runGenericAuthMCPInvokeTest(t, privateKey)

	// Run Advanced Tool Tests
	runAdvancedHTTPMCPInvokeTest(t)

	// Run Query Parameter Tests
	runQueryParamMCPInvokeTest(t)

	// Use shared helper for standard database tools
	t.Run("use shared RunMCPToolInvokeTest", func(t *testing.T) {
		tests.RunMCPToolInvokeTest(t, `"hello world"`,
			tests.WithMyToolId3NameAliceWant(`{"id":1,"name":"Alice"}`),
			tests.WithMyToolById4Want(`{"id":4,"name":null}`),
		)
	})
}

func runGenericAuthMCPInvokeTest(t *testing.T, privateKey *rsa.PrivateKey) {
	// Generic Auth Success
	t.Run("invoke generic auth tool with valid token", func(t *testing.T) {
		// Generate valid token
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"aud":   "test-audience",
			"scope": "read:files",
			"sub":   "test-user",
			"exp":   time.Now().Add(time.Hour).Unix(),
		})
		token.Header["kid"] = "test-key-id"
		signedString, err := token.SignedString(privateKey)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		headers := map[string]string{"my-generic-auth_token": signedString}
		statusCode, mcpResp, err := tests.InvokeMCPTool(t, "my-auth-required-generic-tool", map[string]any{}, headers)
		if err != nil {
			t.Fatalf("native error executing %s: %s", "my-auth-required-generic-tool", err)
		}
		if statusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", statusCode)
		}
		if mcpResp.Result.IsError {
			t.Fatalf("expected success, got error result: %v", mcpResp.Result)
		}
	})

	// Auth Failure: Invoke generic auth tool without token
	t.Run("invoke generic auth tool without token", func(t *testing.T) {
		statusCode, _, err := tests.InvokeMCPTool(t, "my-auth-required-generic-tool", map[string]any{}, nil)
		if err != nil {
			t.Fatalf("native error executing %s: %s", "my-auth-required-generic-tool", err)
		}
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", statusCode)
		}
	})
}

func runQueryParamMCPInvokeTest(t *testing.T) {
	// Query Parameter Variations: Tests with optional parameters omitted or nil
	t.Run("invoke query-param-tool optional omitted", func(t *testing.T) {
		arguments := map[string]any{"reqId": "test1"}
		tests.RunMCPCustomToolCallMethod(t, "my-query-param-tool", arguments, `"reqId=test1"`)
	})

	t.Run("invoke query-param-tool some optional nil", func(t *testing.T) {
		arguments := map[string]any{"reqId": "test2", "page": "5", "filter": nil}
		tests.RunMCPCustomToolCallMethod(t, "my-query-param-tool", arguments, `"page=5\u0026reqId=test2"`) // 'filter' omitted!
	})

	t.Run("invoke query-param-tool some optional absent", func(t *testing.T) {
		arguments := map[string]any{"reqId": "test2", "page": "5"}
		tests.RunMCPCustomToolCallMethod(t, "my-query-param-tool", arguments, `"page=5\u0026reqId=test2"`) // 'filter' omitted!
	})

	t.Run("invoke query-param-tool required param nil", func(t *testing.T) {
		statusCode, mcpResp, err := tests.InvokeMCPTool(t, "my-query-param-tool", map[string]any{"reqId": nil, "page": "1"}, nil)
		if err != nil {
			t.Fatalf("native error executing %s: %s", "my-query-param-tool", err)
		}
		if statusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", statusCode)
		}
		tests.AssertMCPError(t, mcpResp, "required")
	})
}

func runAdvancedHTTPMCPInvokeTest(t *testing.T) {
	// Mock Server Error: Invoke tool with parameters that cause the mock server to return 400
	t.Run("invoke my-advanced-tool with wrong params causing mock 400", func(t *testing.T) {
		arguments := map[string]any{
			"animalArray":    []any{"rabbit", "ostrich", "whale"},
			"id":             4, // Expected 3 in mock!
			"path":           "tool3",
			"country":        "US",
			"X-Other-Header": "test",
		}
		statusCode, mcpResp, err := tests.InvokeMCPTool(t, "my-advanced-tool", arguments, nil)
		if err != nil {
			t.Fatalf("native error executing %s: %s", "my-advanced-tool", err)
		}
		if statusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", statusCode)
		}
		tests.AssertMCPError(t, mcpResp, "unexpected status code")
	})

	// Advanced Tool Success
	t.Run("invoke my-advanced-tool successfully", func(t *testing.T) {
		arguments := map[string]any{
			"animalArray":    []any{"rabbit", "ostrich", "whale"},
			"id":             3,
			"path":           "tool3",
			"country":        "US",
			"X-Other-Header": "test",
		}
		tests.RunMCPCustomToolCallMethod(t, "my-advanced-tool", arguments, `"hello world"`)
	})
}
