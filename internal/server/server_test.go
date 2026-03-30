// Copyright 2024 Google LLC
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

package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/auth/generic"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/alloydbpg"
	"github.com/googleapis/genai-toolbox/internal/telemetry"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
)

func TestServe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr, port := "127.0.0.1", 5000
	cfg := server.ServerConfig{
		Version:      "0.0.0",
		Address:      addr,
		Port:         port,
		AllowedHosts: []string{"*"},
	}

	otelShutdown, err := telemetry.SetupOTel(ctx, "0.0.0", "", false, "toolbox")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	defer func() {
		err := otelShutdown(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}()

	testLogger, err := log.NewStdLogger(os.Stdout, os.Stderr, "info")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	ctx = util.WithLogger(ctx, testLogger)

	instrumentation, err := telemetry.CreateTelemetryInstrumentation(cfg.Version)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	ctx = util.WithInstrumentation(ctx, instrumentation)

	s, err := server.NewServer(ctx, cfg)
	if err != nil {
		t.Fatalf("unable to initialize server: %v", err)
	}

	err = s.Listen(ctx)
	if err != nil {
		t.Fatalf("unable to start server: %v", err)
	}

	// start server in background
	errCh := make(chan error)
	go func() {
		defer close(errCh)

		err = s.Serve(ctx)
		if err != nil {
			errCh <- err
		}
	}()

	url := fmt.Sprintf("http://%s:%d/", addr, port)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("error when sending a request: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("response status code is not 200")
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error reading from request body: %s", err)
	}
	if got := string(raw); strings.Contains(got, "0.0.0") {
		t.Fatalf("version missing from output: %q", got)
	}
}

func TestUpdateServer(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("error setting up logger: %s", err)
	}

	addr, port := "127.0.0.1", 5000
	cfg := server.ServerConfig{
		Version: "0.0.0",
		Address: addr,
		Port:    port,
	}

	instrumentation, err := telemetry.CreateTelemetryInstrumentation(cfg.Version)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	ctx = util.WithInstrumentation(ctx, instrumentation)

	s, err := server.NewServer(ctx, cfg)
	if err != nil {
		t.Fatalf("error setting up server: %s", err)
	}

	newSources := map[string]sources.Source{
		"example-source": &alloydbpg.Source{
			Config: alloydbpg.Config{
				Name: "example-alloydb-source",
				Type: "alloydb-postgres",
			},
		},
	}
	newAuth := map[string]auth.AuthService{"example-auth": nil}
	newEmbeddingModels := map[string]embeddingmodels.EmbeddingModel{"example-model": nil}
	newTools := map[string]tools.Tool{"example-tool": nil}
	newToolsets := map[string]tools.Toolset{
		"example-toolset": {
			ToolsetConfig: tools.ToolsetConfig{
				Name: "example-toolset",
			},
			Tools: []*tools.Tool{},
		},
	}
	newPrompts := map[string]prompts.Prompt{"example-prompt": nil}
	newPromptsets := map[string]prompts.Promptset{
		"example-promptset": {
			PromptsetConfig: prompts.PromptsetConfig{
				Name: "example-promptset",
			},
			Prompts: []*prompts.Prompt{},
		},
	}
	s.ResourceMgr.SetResources(newSources, newAuth, newEmbeddingModels, newTools, newToolsets, newPrompts, newPromptsets)
	if err != nil {
		t.Errorf("error updating server: %s", err)
	}

	gotSource, _ := s.ResourceMgr.GetSource("example-source")
	if diff := cmp.Diff(gotSource, newSources["example-source"]); diff != "" {
		t.Errorf("error updating server, sources (-want +got):\n%s", diff)
	}

	gotAuthService, _ := s.ResourceMgr.GetAuthService("example-auth")
	if diff := cmp.Diff(gotAuthService, newAuth["example-auth"]); diff != "" {
		t.Errorf("error updating server, authServices (-want +got):\n%s", diff)
	}

	gotTool, _ := s.ResourceMgr.GetTool("example-tool")
	if diff := cmp.Diff(gotTool, newTools["example-tool"]); diff != "" {
		t.Errorf("error updating server, tools (-want +got):\n%s", diff)
	}

	gotToolset, _ := s.ResourceMgr.GetToolset("example-toolset")
	if diff := cmp.Diff(gotToolset, newToolsets["example-toolset"]); diff != "" {
		t.Errorf("error updating server, toolset (-want +got):\n%s", diff)
	}

	gotPrompt, _ := s.ResourceMgr.GetPrompt("example-prompt")
	if diff := cmp.Diff(gotPrompt, newPrompts["example-prompt"]); diff != "" {
		t.Errorf("error updating server, prompts (-want +got):\n%s", diff)
	}

	gotPromptset, _ := s.ResourceMgr.GetPromptset("example-promptset")
	if diff := cmp.Diff(gotPromptset, newPromptsets["example-promptset"]); diff != "" {
		t.Errorf("error updating server, promptset (-want +got):\n%s", diff)
	}
}

func TestNameValidation(t *testing.T) {
	testCases := []struct {
		desc         string
		resourceName string
		errStr       string
	}{
		{
			desc:         "names with 0 length",
			resourceName: "",
			errStr:       "resource name SHOULD be between 1 and 128 characters in length (inclusive)",
		},
		{
			desc:         "names with allowed length",
			resourceName: "foo",
		},
		{
			desc:         "names with 128 length",
			resourceName: strings.Repeat("a", 128),
		},
		{
			desc:         "names with more than 128 length",
			resourceName: strings.Repeat("a", 129),
			errStr:       "resource name SHOULD be between 1 and 128 characters in length (inclusive)",
		},
		{
			desc:         "names with space",
			resourceName: "foo bar",
			errStr:       "invalid character for resource name; only uppercase and lowercase ASCII letters (A-Z, a-z), digits (0-9), underscore (_), hyphen (-), and dot (.) is allowed",
		},
		{
			desc:         "names with commas",
			resourceName: "foo,bar",
			errStr:       "invalid character for resource name; only uppercase and lowercase ASCII letters (A-Z, a-z), digits (0-9), underscore (_), hyphen (-), and dot (.) is allowed",
		},
		{
			desc:         "names with other special character",
			resourceName: "foo!",
			errStr:       "invalid character for resource name; only uppercase and lowercase ASCII letters (A-Z, a-z), digits (0-9), underscore (_), hyphen (-), and dot (.) is allowed",
		},
		{
			desc:         "names with allowed special character",
			resourceName: "foo_.-bar6",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := server.NameValidation(tc.resourceName)
			if err != nil {
				if tc.errStr != err.Error() {
					t.Fatalf("unexpected error: %s", err)
				}
			}
			if err == nil && tc.errStr != "" {
				t.Fatalf("expect error: %s", tc.errStr)
			}
		})
	}
}

func TestPRMEndpoint(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup telemetry and logging
	otelShutdown, err := telemetry.SetupOTel(ctx, "0.0.0", "", false, "toolbox")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	defer func() {
		if err := otelShutdown(ctx); err != nil {
			t.Fatalf("unexpected error shutting down otel: %s", err)
		}
	}()

	testLogger, err := log.NewStdLogger(os.Stdout, os.Stderr, "info")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	ctx = util.WithLogger(ctx, testLogger)

	instrumentation, err := telemetry.CreateTelemetryInstrumentation("0.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	ctx = util.WithInstrumentation(ctx, instrumentation)

	// Create a mock OIDC server to bypass JWKS discovery during init
	mockOIDC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"issuer": "http://%s", "jwks_uri": "http://%s/jwks"}`, r.Host, r.Host)
			return
		}
		if r.URL.Path == "/jwks" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"keys": []}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockOIDC.Close()

	// Configure the server
	addr, port := "127.0.0.1", 5001
	cfg := server.ServerConfig{
		Version:      "0.0.0",
		Address:      addr,
		Port:         port,
		ToolboxUrl:   "https://my-toolbox.example.com",
		AllowedHosts: []string{"*"},
		AuthServiceConfigs: map[string]auth.AuthServiceConfig{
			"generic1": generic.Config{
				Name:                "generic1",
				Type:                generic.AuthServiceType,
				McpEnabled:          true,
				AuthorizationServer: mockOIDC.URL, // Injecting the mock server URL here
				ScopesRequired:      []string{"read", "write"},
			},
		},
	}

	// Initialize and start the server
	s, err := server.NewServer(ctx, cfg)
	if err != nil {
		t.Fatalf("unable to initialize server: %v", err)
	}

	if err := s.Listen(ctx); err != nil {
		t.Fatalf("unable to start server: %v", err)
	}

	errCh := make(chan error)
	go func() {
		defer close(errCh)
		if err := s.Serve(ctx); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer func() {
		if err := s.Shutdown(ctx); err != nil {
			t.Errorf("failed to cleanly shutdown server: %v", err)
		}
	}()

	// Test the PRM endpoint
	url := fmt.Sprintf("http://%s:%d/.well-known/oauth-protected-resource", addr, port)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("error when sending a request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unexpected error reading body: %s", err)
	}

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unexpected error unmarshalling body: %s", err)
	}

	want := map[string]any{
		"resource": "https://my-toolbox.example.com",
		"authorization_servers": []any{
			mockOIDC.URL,
		},
		"scopes_supported":         []any{"read", "write"},
		"bearer_methods_supported": []any{"header"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected PRM response:\ngot  %+v\nwant %+v", got, want)
	}
}

func TestPRMOverride(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup a temporary PRM file
	prmContent := `{
		"resource": "https://override.example.com",
		"authorization_servers": ["https://auth.example.com"],
		"scopes_supported": ["read", "write"],
		"bearer_methods_supported": ["header"]
	}`
	tmpFile, err := os.CreateTemp("", "prm-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	if err := os.WriteFile(tmpFile.Name(), []byte(prmContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Setup Logging and Instrumentation (Using Discard to act as Noop)
	testLogger, err := log.NewStdLogger(io.Discard, io.Discard, "info")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	ctx = util.WithLogger(ctx, testLogger)

	instrumentation, err := telemetry.CreateTelemetryInstrumentation("0.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	ctx = util.WithInstrumentation(ctx, instrumentation)

	// Configure the server with the Override Flag
	addr, port := "127.0.0.1", 5002
	cfg := server.ServerConfig{
		Version:      "0.0.0",
		Address:      addr,
		Port:         port,
		McpPrmFile:   tmpFile.Name(),
		AllowedHosts: []string{"*"},
	}

	// Initialize and Start the Server
	s, err := server.NewServer(ctx, cfg)
	if err != nil {
		t.Fatalf("unable to initialize server: %v", err)
	}

	if err := s.Listen(ctx); err != nil {
		t.Fatalf("unable to start listener: %v", err)
	}

	go func() {
		if err := s.Serve(ctx); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server serve error: %v\n", err)
		}
	}()
	defer func() {
		if err := s.Shutdown(ctx); err != nil {
			t.Errorf("failed to cleanly shutdown server: %v", err)
		}
	}()

	// Perform the request to the well-known endpoint
	url := fmt.Sprintf("http://%s:%d/.well-known/oauth-protected-resource", addr, port)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("error when sending request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error reading body: %s", err)
	}

	// Verification
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("invalid json response: %s", err)
	}

	if got["resource"] != "https://override.example.com" {
		t.Errorf("expected resource 'https://override.example.com', got '%v'", got["resource"])
	}
}

// TestLegacyAPIGone verifies that requests to legacy /api/* endpoints return 410 Gone.
func TestLegacyAPIGone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup Logging and Instrumentation (Using Discard to act as Noop)
	testLogger, err := log.NewStdLogger(io.Discard, io.Discard, "info")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	ctx = util.WithLogger(ctx, testLogger)

	instrumentation, err := telemetry.CreateTelemetryInstrumentation("0.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	ctx = util.WithInstrumentation(ctx, instrumentation)

	// Configure the server (EnableAPI defaults to false)
	addr, port := "127.0.0.1", 5003
	cfg := server.ServerConfig{
		Version:      "0.0.0",
		Address:      addr,
		Port:         port,
		AllowedHosts: []string{"*"},
	}

	// Initialize and Start the Server
	s, err := server.NewServer(ctx, cfg)
	if err != nil {
		t.Fatalf("unable to initialize server: %v", err)
	}

	if err := s.Listen(ctx); err != nil {
		t.Fatalf("unable to start listener: %v", err)
	}

	go func() {
		if err := s.Serve(ctx); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server serve error: %v\n", err)
		}
	}()
	defer func() {
		if err := s.Shutdown(ctx); err != nil {
			t.Errorf("failed to cleanly shutdown server: %v", err)
		}
	}()

	// Perform the request to a legacy endpoint
	url := fmt.Sprintf("http://%s:%d/api/tool/list", addr, port)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("error when sending request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusGone {
		t.Fatalf("expected status 410 (Gone), got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error reading body: %s", err)
	}

	want := "/api native endpoints are disabled by default. Please use the standard /mcp JSON-RPC endpoint"
	if !strings.Contains(string(body), want) {
		t.Errorf("expected response body to contain %q, got %q", want, string(body))
	}
}
