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

package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/googleapis/mcp-toolbox/internal/log"
	"github.com/googleapis/mcp-toolbox/internal/sources"
)

// fakeOAuthProxySource is a minimal sources.OAuthProxySource used only by
// this test file. It intentionally does not model any real source type.
type fakeOAuthProxySource struct {
	clientOAuthEnabled bool
	authEndpoint       string
	tokenEndpoint      string
}

var (
	_ sources.Source           = fakeOAuthProxySource{}
	_ sources.OAuthProxySource = fakeOAuthProxySource{}
)

func (f fakeOAuthProxySource) SourceType() string             { return "fake" }
func (f fakeOAuthProxySource) ToConfig() sources.SourceConfig { return nil }
func (f fakeOAuthProxySource) IsReadOnly() bool               { return true }
func (f fakeOAuthProxySource) ClientOAuthEnabled() bool       { return f.clientOAuthEnabled }
func (f fakeOAuthProxySource) OAuthAuthorizationEndpoint() string {
	return f.authEndpoint
}
func (f fakeOAuthProxySource) OAuthTokenEndpoint() string {
	return f.tokenEndpoint
}

// fakeOAuthProxySourceWithCapabilities additionally implements
// sources.OAuthProxyCapabilities, to test that a source can override the
// default advertised OAuth capabilities.
type fakeOAuthProxySourceWithCapabilities struct {
	fakeOAuthProxySource
	responseTypes    []string
	grantTypes       []string
	challengeMethods []string
}

var _ sources.OAuthProxyCapabilities = fakeOAuthProxySourceWithCapabilities{}

func (f fakeOAuthProxySourceWithCapabilities) OAuthResponseTypesSupported() []string {
	return f.responseTypes
}
func (f fakeOAuthProxySourceWithCapabilities) OAuthGrantTypesSupported() []string {
	return f.grantTypes
}
func (f fakeOAuthProxySourceWithCapabilities) OAuthCodeChallengeMethodsSupported() []string {
	return f.challengeMethods
}

// fakeNonProxySource implements sources.Source but not
// sources.OAuthProxySource, to verify the type assertion correctly skips it.
type fakeNonProxySource struct{}

var _ sources.Source = fakeNonProxySource{}

func (f fakeNonProxySource) SourceType() string             { return "fake-non-proxy" }
func (f fakeNonProxySource) ToConfig() sources.SourceConfig { return nil }
func (f fakeNonProxySource) IsReadOnly() bool               { return true }

func testLogger(t *testing.T) log.Logger {
	t.Helper()
	l, err := log.NewStdLogger(os.Stdout, os.Stderr, "debug")
	if err != nil {
		t.Fatalf("unable to initialize test logger: %s", err)
	}
	return l
}

func TestCollectOAuthProxySources_None(t *testing.T) {
	matches := collectOAuthProxySources(map[string]sources.Source{
		"db": fakeNonProxySource{},
	})
	if len(matches) != 0 {
		t.Fatalf("expected no matches, got %v", matches)
	}
}

func TestCollectOAuthProxySources_IgnoresDisabled(t *testing.T) {
	matches := collectOAuthProxySources(map[string]sources.Source{
		"looker": fakeOAuthProxySource{clientOAuthEnabled: false, authEndpoint: "https://a/auth", tokenEndpoint: "https://a/token"},
	})
	if len(matches) != 0 {
		t.Fatalf("expected no matches for a source with client OAuth disabled, got %v", matches)
	}
}

func TestCollectOAuthProxySources_MultipleAllowed(t *testing.T) {
	// This is the key generalization regression guard: unlike an earlier,
	// narrower version of this mechanism, MULTIPLE sources with client-side
	// OAuth enabled must be allowed to coexist, not rejected.
	matches := collectOAuthProxySources(map[string]sources.Source{
		"other-db": fakeNonProxySource{},
		"looker-a": fakeOAuthProxySource{clientOAuthEnabled: true, authEndpoint: "https://a/auth", tokenEndpoint: "https://a/token"},
		"looker-b": fakeOAuthProxySource{clientOAuthEnabled: true, authEndpoint: "https://b/auth", tokenEndpoint: "https://b/token"},
	})
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d: %v", len(matches), matches)
	}
	if _, ok := matches["looker-a"]; !ok {
		t.Errorf("expected looker-a in matches")
	}
	if _, ok := matches["looker-b"]; !ok {
		t.Errorf("expected looker-b in matches")
	}
}

func TestMountOAuthProxyRoutes_NoOpWhenNoSourceNeedsIt(t *testing.T) {
	r := chi.NewRouter()
	err := mountOAuthProxyRoutes(context.Background(), r, map[string]sources.Source{
		"db": fakeNonProxySource{},
	}, "", testLogger(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/oauth-proxy/db/.well-known/oauth-authorization-server")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Errorf("expected no proxy route to be mounted when no source needs it, got 200")
	}
}

func TestMountOAuthProxyRoutes_ErrorsOnEmptyEndpoint(t *testing.T) {
	r := chi.NewRouter()
	err := mountOAuthProxyRoutes(context.Background(), r, map[string]sources.Source{
		"looker": fakeOAuthProxySource{clientOAuthEnabled: true, authEndpoint: "", tokenEndpoint: "https://looker.example.com/oauth/token"},
	}, "", testLogger(t))
	if err == nil {
		t.Fatal("expected an error when the matched source returns an empty authorization endpoint")
	}
}

func TestMountOAuthProxyRoutes_NamespacesMultipleSources(t *testing.T) {
	r := chi.NewRouter()
	err := mountOAuthProxyRoutes(context.Background(), r, map[string]sources.Source{
		"looker-a": fakeOAuthProxySource{clientOAuthEnabled: true, authEndpoint: "https://a.example.com/auth", tokenEndpoint: "https://a.example.com/oauth/token"},
		"looker-b": fakeOAuthProxySource{clientOAuthEnabled: true, authEndpoint: "https://b.example.com/auth", tokenEndpoint: "https://b.example.com/oauth/token"},
	}, "", testLogger(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	for name, wantAuthEndpoint := range map[string]string{
		"looker-a": "https://a.example.com/auth",
		"looker-b": "https://b.example.com/auth",
	} {
		resp, err := http.Get(ts.URL + "/oauth-proxy/" + name + "/.well-known/oauth-authorization-server")
		if err != nil {
			t.Fatalf("GET failed for %s: %v", name, err)
		}
		var metadata oauthAuthorizationServerMetadata
		if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
			t.Fatalf("failed to decode response for %s: %v", name, err)
		}
		resp.Body.Close()
		if metadata.AuthorizationEndpoint != wantAuthEndpoint {
			t.Errorf("%s: got authorization_endpoint %q, want %q", name, metadata.AuthorizationEndpoint, wantAuthEndpoint)
		}
		wantTokenEndpoint := ts.URL + "/oauth-proxy/" + name + "/token"
		if metadata.TokenEndpoint != wantTokenEndpoint {
			t.Errorf("%s: got token_endpoint %q, want %q", name, metadata.TokenEndpoint, wantTokenEndpoint)
		}
	}
}

func TestOAuthDiscoveryHandler_PointsTokenEndpointBackAtProxy(t *testing.T) {
	r := chi.NewRouter()
	err := mountOAuthProxyRoutes(context.Background(), r, map[string]sources.Source{
		"looker": fakeOAuthProxySource{clientOAuthEnabled: true, authEndpoint: "https://looker.example.com/auth", tokenEndpoint: "https://looker.example.com/oauth/token"},
	}, "", testLogger(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/oauth-proxy/looker/.well-known/oauth-authorization-server")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want 200", resp.StatusCode)
	}

	var metadata oauthAuthorizationServerMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if metadata.AuthorizationEndpoint != "https://looker.example.com/auth" {
		t.Errorf("got authorization_endpoint %q, want it to point directly at the authorization server", metadata.AuthorizationEndpoint)
	}
	// This is the important regression guard: the advertised token_endpoint
	// must point back at THIS toolbox instance's proxy path, not at the
	// authorization server's real token endpoint — otherwise a
	// standards-compliant OAuth client would read this document and then
	// hit the authorization server's CORS-less token endpoint directly,
	// reintroducing the exact browser CORS failure this proxy exists to
	// work around.
	wantTokenEndpoint := ts.URL + "/oauth-proxy/looker/token"
	if metadata.TokenEndpoint != wantTokenEndpoint {
		t.Errorf("got token_endpoint %q, want %q (must point back at the toolbox's own proxy, not the authorization server)", metadata.TokenEndpoint, wantTokenEndpoint)
	}
	if len(metadata.ResponseTypesSupported) != 1 || metadata.ResponseTypesSupported[0] != "code" {
		t.Errorf("got response_types_supported %v, want default [\"code\"]", metadata.ResponseTypesSupported)
	}
}

func TestOAuthDiscoveryHandler_CustomCapabilitiesOverrideDefaults(t *testing.T) {
	r := chi.NewRouter()
	src := fakeOAuthProxySourceWithCapabilities{
		fakeOAuthProxySource: fakeOAuthProxySource{
			clientOAuthEnabled: true,
			authEndpoint:       "https://looker.example.com/auth",
			tokenEndpoint:      "https://looker.example.com/oauth/token",
		},
		responseTypes:    []string{"code", "token"},
		grantTypes:       []string{"authorization_code", "refresh_token"},
		challengeMethods: []string{"plain", "S256"},
	}
	err := mountOAuthProxyRoutes(context.Background(), r, map[string]sources.Source{
		"looker": src,
	}, "", testLogger(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ts := httptest.NewServer(r)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/oauth-proxy/looker/.well-known/oauth-authorization-server")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	var metadata oauthAuthorizationServerMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if strings.Join(metadata.ResponseTypesSupported, ",") != "code,token" {
		t.Errorf("got response_types_supported %v, want overridden [\"code\",\"token\"]", metadata.ResponseTypesSupported)
	}
	if strings.Join(metadata.GrantTypesSupported, ",") != "authorization_code,refresh_token" {
		t.Errorf("got grant_types_supported %v, want overridden set", metadata.GrantTypesSupported)
	}
	if strings.Join(metadata.CodeChallengeMethodsSupported, ",") != "plain,S256" {
		t.Errorf("got code_challenge_methods_supported %v, want overridden set", metadata.CodeChallengeMethodsSupported)
	}
}

// newFakeAuthServer starts an httptest server standing in for the OAuth
// authorization server's own token endpoint, capturing what it receives so
// tests can assert on the proxy's forwarding behavior.
func newFakeAuthServer(t *testing.T, statusCode int, respBody string) (*httptest.Server, *capturedRequest) {
	t.Helper()
	captured := &capturedRequest{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.method = r.Method
		captured.path = r.URL.Path
		captured.contentType = r.Header.Get("Content-Type")
		body := map[string]string{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		captured.jsonBody = body
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(respBody))
	}))
	return ts, captured
}

type capturedRequest struct {
	method      string
	path        string
	contentType string
	jsonBody    map[string]string
}

func TestOAuthTokenProxyHandler_FormEncodedIsReEncodedAsJSON(t *testing.T) {
	authTS, captured := newFakeAuthServer(t, http.StatusOK, `{"access_token":"abc"}`)
	defer authTS.Close()

	r := chi.NewRouter()
	if err := mountOAuthProxyRoutes(context.Background(), r, map[string]sources.Source{
		"looker": fakeOAuthProxySource{clientOAuthEnabled: true, authEndpoint: authTS.URL + "/auth", tokenEndpoint: authTS.URL + "/oauth/token"},
	}, "", testLogger(t)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	proxyTS := httptest.NewServer(r)
	defer proxyTS.Close()

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", "the-code")
	form.Set("code_verifier", "the-verifier")
	form.Set("client_id", "the-client-id")
	form.Set("redirect_uri", "https://client.example.com/callback")

	resp, err := http.Post(proxyTS.URL+"/oauth-proxy/looker/token", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want 200", resp.StatusCode)
	}
	if captured.method != http.MethodPost {
		t.Errorf("upstream received method %q, want POST", captured.method)
	}
	if captured.path != "/oauth/token" {
		t.Errorf("upstream received path %q, want /oauth/token", captured.path)
	}
	if captured.contentType != "application/json" {
		t.Errorf("upstream received Content-Type %q, want application/json (form body should be re-encoded)", captured.contentType)
	}
	if captured.jsonBody["grant_type"] != "authorization_code" || captured.jsonBody["code"] != "the-code" || captured.jsonBody["code_verifier"] != "the-verifier" {
		t.Errorf("upstream did not receive the expected re-encoded fields: %+v", captured.jsonBody)
	}
}

func TestOAuthTokenProxyHandler_JSONPassthrough(t *testing.T) {
	authTS, captured := newFakeAuthServer(t, http.StatusOK, `{"access_token":"abc"}`)
	defer authTS.Close()

	r := chi.NewRouter()
	if err := mountOAuthProxyRoutes(context.Background(), r, map[string]sources.Source{
		"looker": fakeOAuthProxySource{clientOAuthEnabled: true, authEndpoint: authTS.URL + "/auth", tokenEndpoint: authTS.URL + "/oauth/token"},
	}, "", testLogger(t)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	proxyTS := httptest.NewServer(r)
	defer proxyTS.Close()

	resp, err := http.Post(proxyTS.URL+"/oauth-proxy/looker/token", "application/json", strings.NewReader(`{"grant_type":"refresh_token","refresh_token":"rt-123"}`))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want 200", resp.StatusCode)
	}
	if captured.contentType != "application/json" {
		t.Errorf("upstream received Content-Type %q, want application/json", captured.contentType)
	}
	if captured.jsonBody["refresh_token"] != "rt-123" {
		t.Errorf("upstream did not receive the expected JSON body passed through as-is: %+v", captured.jsonBody)
	}
}

func TestOAuthTokenProxyHandler_UpstreamUnreachableReturnsBadGateway(t *testing.T) {
	// Close immediately so the address is guaranteed unreachable.
	authTS, _ := newFakeAuthServer(t, http.StatusOK, `{}`)
	authTS.Close()

	r := chi.NewRouter()
	if err := mountOAuthProxyRoutes(context.Background(), r, map[string]sources.Source{
		"looker": fakeOAuthProxySource{clientOAuthEnabled: true, authEndpoint: authTS.URL + "/auth", tokenEndpoint: authTS.URL + "/oauth/token"},
	}, "", testLogger(t)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	proxyTS := httptest.NewServer(r)
	defer proxyTS.Close()

	resp, err := http.Post(proxyTS.URL+"/oauth-proxy/looker/token", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("got status %d, want 502 when the authorization server is unreachable", resp.StatusCode)
	}
}

// newTLSServerWithFreshCert starts an httptest TLS server with its own
// freshly generated, independent self-signed certificate. httptest.NewTLSServer
// alone reuses the same shared built-in test certificate across every call,
// so two servers started that way are indistinguishable from a CA-trust
// perspective — that would make a test for "a CA that doesn't match" pass
// vacuously. Generating a distinct cert per server makes the two servers
// genuinely unrelated for TLS trust purposes.
func newTLSServerWithFreshCert(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create test certificate: %v", err)
	}

	ts := httptest.NewUnstartedServer(handler)
	ts.TLS = &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{der},
			PrivateKey:  priv,
		}},
	}
	ts.StartTLS()
	return ts
}

func TestNewOAuthProxyHTTPClient_DefaultVerificationRejectsUntrustedCert(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client, err := newOAuthProxyHTTPClient("")
	if err != nil {
		t.Fatalf("newOAuthProxyHTTPClient(\"\") returned error: %v", err)
	}

	_, err = client.Get(ts.URL)
	if err == nil {
		t.Fatal("expected a TLS verification error for an untrusted self-signed cert, got nil")
	}
}

func TestNewOAuthProxyHTTPClient_CustomCATrustsMatchingServer(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	caCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: ts.Certificate().Raw,
	})
	dir := t.TempDir()
	caFile := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(caFile, caCertPEM, 0o600); err != nil {
		t.Fatalf("failed to write test CA file: %v", err)
	}

	client, err := newOAuthProxyHTTPClient(caFile)
	if err != nil {
		t.Fatalf("newOAuthProxyHTTPClient(%q) returned error: %v", caFile, err)
	}

	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("request with matching custom CA should succeed, got error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}
}

func TestNewOAuthProxyHTTPClient_CustomCAStillRejectsUnrelatedServer(t *testing.T) {
	trustedTS := newTLSServerWithFreshCert(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer trustedTS.Close()
	unrelatedTS := newTLSServerWithFreshCert(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer unrelatedTS.Close()

	caCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: trustedTS.Certificate().Raw,
	})
	dir := t.TempDir()
	caFile := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(caFile, caCertPEM, 0o600); err != nil {
		t.Fatalf("failed to write test CA file: %v", err)
	}

	client, err := newOAuthProxyHTTPClient(caFile)
	if err != nil {
		t.Fatalf("newOAuthProxyHTTPClient(%q) returned error: %v", caFile, err)
	}

	_, err = client.Get(unrelatedTS.URL)
	if err == nil {
		t.Fatal("expected a TLS verification error against a server signed by an unrelated CA, got nil")
	}
}

func TestNewOAuthProxyHTTPClient_MissingCAFileReturnsError(t *testing.T) {
	_, err := newOAuthProxyHTTPClient(filepath.Join(t.TempDir(), "does-not-exist.pem"))
	if err == nil {
		t.Fatal("expected an error for a nonexistent CA cert file, got nil")
	}
}

func TestNewOAuthProxyHTTPClient_InvalidCAFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	badFile := filepath.Join(dir, "not-a-cert.pem")
	if err := os.WriteFile(badFile, []byte("this is not a certificate"), 0o600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := newOAuthProxyHTTPClient(badFile)
	if err == nil {
		t.Fatal("expected an error for a malformed CA cert file, got nil")
	}
}
