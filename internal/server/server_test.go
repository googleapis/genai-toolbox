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
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/auth"
	"github.com/googleapis/mcp-toolbox/internal/auth/generic"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/log"
	"github.com/googleapis/mcp-toolbox/internal/prompts"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/sources/alloydbpg"
	"github.com/googleapis/mcp-toolbox/internal/telemetry"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
)

// Helper function to create temporary self-signed certs for the test
func generateTestCerts(t *testing.T) (string, string, func()) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"Test Co"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	// Create temp files
	certFile, err := os.CreateTemp("", "cert.*.pem")
	if err != nil {
		t.Fatalf("failed to create temp cert file: %v", err)
	}

	keyFile, err := os.CreateTemp("", "key.*.pem")
	if err != nil {
		t.Fatalf("failed to create temp key file: %v", err)
	}

	// Check the error return values for pem.Encode
	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		t.Fatalf("failed to encode certificate: %v", err)
	}

	if err := pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}); err != nil {
		t.Fatalf("failed to encode key: %v", err)
	}

	certFile.Close()
	keyFile.Close()

	cleanup := func() {
		os.Remove(certFile.Name())
		os.Remove(keyFile.Name())
	}

	return certFile.Name(), keyFile.Name(), cleanup
}

func TestServe(t *testing.T) {
	certFile, keyFile, cleanupCerts := generateTestCerts(t)
	defer cleanupCerts()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	tests := []struct {
		name string
		cert string
		key  string
		addr string
		port int
	}{
		{
			name: "HTTP mode",
			addr: "127.0.0.1",
			port: 5001,
		},
		{
			name: "HTTPS mode",
			cert: certFile,
			key:  keyFile,
			addr: "127.0.0.1",
			port: 5002,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := server.ServerConfig{
				Version:      "0.0.0",
				Address:      tt.addr,
				Port:         tt.port,
				AllowedHosts: []string{"*"},
			}

			instrumentation, err := telemetry.CreateTelemetryInstrumentation(cfg.Version)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			ctx = util.WithInstrumentation(ctx, instrumentation)

			s, err := server.NewServer(ctx, cfg)
			if err != nil {
				t.Fatalf("unable to initialize server: %v", err)
			}

			err = s.Listen(ctx, tt.cert, tt.key)
			if err != nil {
				t.Fatalf("unable to start server: %v", err)
			}

			// start server in background
			go func() {
				if err := s.Serve(ctx); err != nil && err != http.ErrServerClosed {
					t.Errorf("server serve error: %v", err)
				}
			}()

			// Setup Client to handle self-signed certs
			client := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			}

			useTLS := tt.cert != "" || tt.key != ""
			protocol := "http"
			if useTLS {
				protocol = "https"
			}

			url := fmt.Sprintf("%s://%s:%d/", protocol, tt.addr, tt.port)
			resp, err := client.Get(url)
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
		})
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
	if diff := cmp.Diff(gotToolset, newToolsets["example-toolset"], cmp.AllowUnexported(tools.Toolset{})); diff != "" {
		t.Errorf("error updating server, toolset (-want +got):\n%s", diff)
	}

	gotPrompt, _ := s.ResourceMgr.GetPrompt("example-prompt")
	if diff := cmp.Diff(gotPrompt, newPrompts["example-prompt"]); diff != "" {
		t.Errorf("error updating server, prompts (-want +got):\n%s", diff)
	}

	gotPromptset, _ := s.ResourceMgr.GetPromptset("example-promptset")
	if diff := cmp.Diff(gotPromptset, newPromptsets["example-promptset"], cmp.AllowUnexported(prompts.Promptset{})); diff != "" {
		t.Errorf("error updating server, promptset (-want +got):\n%s", diff)
	}
}

func TestEndpointSecurityAllowedOrigin(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("error setting up logger: %s", err)
	}

	testCases := []struct {
		desc           string
		allowedOrigins []string
		origin         string
		corsBlocked    bool
	}{
		{
			desc:           "allowed origin all",
			allowedOrigins: []string{"*"},
			origin:         "https://evil.com",
		},
		{
			desc:           "allowed origin trusted with trusted origin",
			allowedOrigins: []string{"https://trusted.com"},
			origin:         "https://trusted.com",
		},
		{
			desc:           "allowed origin trusted with evil origin",
			allowedOrigins: []string{"https://trusted.com"},
			origin:         "https://evil.com",
			corsBlocked:    true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			addr, port := "127.0.0.1", 0
			cfg := server.ServerConfig{
				Version:        "0.0.0",
				Address:        addr,
				Port:           port,
				EnableAPI:      true,
				AllowedOrigins: tc.allowedOrigins,
				AllowedHosts:   []string{"*"},
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

			err = s.Listen(ctx, "", "")
			if err != nil {
				t.Fatalf("unable to start server: %v", err)
			}

			urlAddr := s.Addr()

			// start server in background
			go func() {
				if err := s.Serve(ctx); err != nil && err != http.ErrServerClosed {
					t.Errorf("server serve error: %v", err)
				}
			}()

			// test every endpoints that we support in Toolbox
			endpoints := []struct {
				desc        string
				requestType string
				url         string
			}{
				{
					desc:        "GET api toolset",
					requestType: "GET",
					url:         "/api/toolset",
				},
				{
					desc:        "GET api tool",
					requestType: "GET",
					url:         "/api/tool/tool_one",
				},
				{
					desc:        "POST api tool",
					requestType: "POST",
					url:         "/api/tool/tool_one/invoke",
				},
				{
					desc:        "GET mcp sse",
					requestType: "GET",
					url:         "/mcp/sse",
				},
				{
					desc:        "GET mcp",
					requestType: "GET",
					url:         "/mcp",
				},
				{
					desc:        "POST mcp",
					requestType: "POST",
					url:         "/mcp",
				},
				{
					desc:        "DELETE mcp",
					requestType: "DELETE",
					url:         "/mcp",
				},
			}
			for _, e := range endpoints {
				t.Run(e.desc, func(t *testing.T) {
					url := fmt.Sprintf("http://%s%s", urlAddr, e.url)
					client := &http.Client{}
					req, err := http.NewRequest(e.requestType, url, nil)
					if err != nil {
						t.Fatalf("Failed to create request: %v", err)
					}
					req.Header.Set("Origin", tc.origin)
					resp, err := client.Do(req)
					if err != nil {
						t.Fatalf("Failed to send request: %v", err)
					}
					defer resp.Body.Close()

					gotOrigin := resp.Header.Get("Access-Control-Allow-Origin")
					if !tc.corsBlocked {
						// if cors is not blocked, the origin header should be
						// within allowedOrigins
						if !slices.Contains(tc.allowedOrigins, gotOrigin) {
							t.Errorf(`origin "%s" is not part of allowed origins %s`, gotOrigin, tc.allowedOrigins)
						}
					} else if tc.corsBlocked {
						// if cors is blocked, the origin header should not
						// contain origin
						if gotOrigin == "*" {
							t.Errorf("REGRESSION: Server is forcing a wildcard '*' header!")
						}
						if gotOrigin == tc.origin {
							t.Errorf("server allowed an origin not in the whitelist: %s", gotOrigin)
						}
					}
				})
			}
		})
	}
}

func TestEndpointSecurityAllowedHost(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("error setting up logger: %s", err)
	}

	testCases := []struct {
		desc         string
		allowedHosts []string
		host         string
		wantStatus   int
	}{
		{
			desc:         "allowed hosts all",
			allowedHosts: []string{"*"},
			host:         "evil.com",
			wantStatus:   http.StatusOK,
		},
		{
			desc:         "allowed hosts trusted with trusted host",
			allowedHosts: []string{"trusted.com"},
			host:         "trusted.com",
			wantStatus:   http.StatusOK,
		},
		{
			desc:         "allowed hosts trusted with evil host",
			allowedHosts: []string{"trusted.com"},
			host:         "evil.com",
			wantStatus:   http.StatusForbidden,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			addr, port := "127.0.0.1", 0
			cfg := server.ServerConfig{
				Version:      "0.0.0",
				Address:      addr,
				Port:         port,
				EnableAPI:    true,
				AllowedHosts: tc.allowedHosts,
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

			err = s.Listen(ctx, "", "")
			if err != nil {
				t.Fatalf("unable to start server: %v", err)
			}

			urlAddr := s.Addr()
			_, actualPort, err := net.SplitHostPort(urlAddr)
			if err != nil {
				t.Fatalf("failed to parse server address: %v", err)
			}
			hostWithPort := net.JoinHostPort(tc.host, actualPort)

			// start server in background
			go func() {
				if err := s.Serve(ctx); err != nil && err != http.ErrServerClosed {
					t.Errorf("server serve error: %v", err)
				}
			}()

			// test every endpoints that we support in Toolbox
			endpoints := []struct {
				desc        string
				requestType string
				url         string
				requestErr  int
				errStr      string
			}{
				{
					desc:        "GET api toolset",
					requestType: "GET",
					url:         "/api/toolset",
				},
				{
					desc:        "GET api tool",
					requestType: "GET",
					url:         "/api/tool/tool_one",
					requestErr:  http.StatusNotFound,
					errStr:      "invalid tool name",
				},
				{
					desc:        "POST api tool",
					requestType: "POST",
					url:         "/api/tool/tool_one/invoke",
					requestErr:  http.StatusNotFound,
					errStr:      "invalid tool name",
				},
				{
					desc:        "GET mcp sse",
					requestType: "GET",
					url:         "/mcp/sse",
				},
				{
					desc:        "GET mcp",
					requestType: "GET",
					url:         "/mcp",
					requestErr:  http.StatusMethodNotAllowed,
					errStr:      "toolbox does not support streaming in streamable HTTP transport",
				},
				{
					desc:        "POST mcp",
					requestType: "POST",
					url:         "/mcp",
				},
				{
					desc:        "DELETE mcp",
					requestType: "DELETE",
					url:         "/mcp",
				},
			}
			for _, e := range endpoints {
				t.Run(e.desc, func(t *testing.T) {
					url := fmt.Sprintf("http://%s%s", urlAddr, e.url)
					client := &http.Client{}
					req, err := http.NewRequest(e.requestType, url, nil)
					if err != nil {
						t.Fatalf("Failed to create request: %v", err)
					}
					req.Host = hostWithPort
					resp, err := client.Do(req)
					if err != nil {
						t.Fatalf("Failed to send request: %v", err)
					}
					defer resp.Body.Close()

					if resp.StatusCode != tc.wantStatus {
						bodyBytes, _ := io.ReadAll(resp.Body)
						if resp.StatusCode == e.requestErr {
							if !strings.Contains(string(bodyBytes), e.errStr) {
								t.Fatalf("got err %s, expected error %s", string(bodyBytes), e.errStr)
							}
							return
						}
						t.Fatalf("expected status %d, got %d: %s", tc.wantStatus, resp.StatusCode, string(bodyBytes))
					}
				})
			}
		})
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
	addr, port := "127.0.0.1", 5003
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

	if err := s.Listen(ctx, "", ""); err != nil {
		t.Fatalf("unable to start server: %v", err)
	}

	go func() {
		if err := s.Serve(ctx); err != nil && err != http.ErrServerClosed {
			t.Errorf("server serve error: %v", err)
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
	addr, port := "127.0.0.1", 5004
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

	if err := s.Listen(ctx, "", ""); err != nil {
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
	addr, port := "127.0.0.1", 5005
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

	if err := s.Listen(ctx, "", ""); err != nil {
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

func TestMCPAuthMiddleware(t *testing.T) {
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

	// Setup mock introspection server
	var mockResponse map[string]any
	var mockStatus int
	var mockRawResponse string

	mockOIDC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"issuer": "http://%s", "jwks_uri": "http://%s/jwks", "introspection_endpoint": "http://%s/introspect"}`, r.Host, r.Host, r.Host)
			return
		}
		if r.URL.Path == "/jwks" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"keys": []}`)
			return
		}
		if r.URL.Path == "/introspect" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(mockStatus)
			if mockRawResponse != "" {
				_, _ = w.Write([]byte(mockRawResponse))
			} else {
				_ = json.NewEncoder(w).Encode(mockResponse)
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockOIDC.Close()

	// Configure the server
	addr, port := "127.0.0.1", 5004
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
				AuthorizationServer: mockOIDC.URL,
				ScopesRequired:      []string{"mcp"},
			},
		},
	}

	// Initialize and start the server
	s, err := server.NewServer(ctx, cfg)
	if err != nil {
		t.Fatalf("unable to initialize server: %v", err)
	}

	if err := s.Listen(ctx, "", ""); err != nil {
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

	tests := []struct {
		name           string
		token          string
		setupMock      func()
		wantStatusCode int
	}{
		{
			name:  "valid opaque token",
			token: "valid-token",
			setupMock: func() {
				mockStatus = http.StatusOK
				mockResponse = map[string]any{
					"active": true,
					"scope":  "mcp",
					"aud":    "test-audience",
					"exp":    time.Now().Add(time.Hour).Unix(),
				}
				mockRawResponse = ""
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:  "insufficient scope",
			token: "bad-scope-token",
			setupMock: func() {
				mockStatus = http.StatusOK
				mockResponse = map[string]any{
					"active": true,
					"scope":  "wrong-scope",
					"aud":    "test-audience",
					"exp":    time.Now().Add(time.Hour).Unix(),
				}
				mockRawResponse = ""
			},
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:  "malformed introspection",
			token: "any-token",
			setupMock: func() {
				mockStatus = http.StatusOK
				mockRawResponse = "{invalid json}"
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:  "unreachable introspection",
			token: "any-token",
			setupMock: func() {
				mockOIDC.Close()
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	url := fmt.Sprintf("http://%s:%d/mcp", addr, port)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMock()

			reqBody := []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
			req, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
			req.Header.Set("Authorization", "Bearer "+tc.token)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatusCode {
				t.Errorf("expected status %d, got %d", tc.wantStatusCode, resp.StatusCode)
			}

			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				t.Errorf("expected Content-Type to contain application/json, got %q", contentType)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}

			var jsonResp map[string]any
			if err := json.Unmarshal(body, &jsonResp); err != nil {
				t.Errorf("response body is not valid JSON: %v\nBody: %s", err, string(body))
			}

			if tc.wantStatusCode != http.StatusOK {
				if _, ok := jsonResp["error"]; !ok {
					t.Errorf("expected error field in response, got: %s", string(body))
				}
				if jsonResp["jsonrpc"] != "2.0" {
					t.Errorf("expected jsonrpc 2.0, got: %v", jsonResp["jsonrpc"])
				}
			} else {
				if _, ok := jsonResp["result"]; !ok {
					t.Errorf("expected result field in response, got: %s", string(body))
				}
			}
		})
	}
}
