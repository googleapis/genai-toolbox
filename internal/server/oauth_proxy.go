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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/googleapis/mcp-toolbox/internal/log"
	"github.com/googleapis/mcp-toolbox/internal/sources"
)

// defaultOAuthResponseTypesSupported, defaultOAuthGrantTypesSupported, and
// defaultOAuthCodeChallengeMethodsSupported are the OAuth capabilities
// advertised for a source implementing sources.OAuthProxySource that does
// not also implement sources.OAuthProxyCapabilities.
var (
	defaultOAuthResponseTypesSupported        = []string{"code"}
	defaultOAuthGrantTypesSupported           = []string{"authorization_code"}
	defaultOAuthCodeChallengeMethodsSupported = []string{"S256"}
)

// oauthAuthorizationServerMetadata is a minimal RFC 8414 OAuth 2.0
// Authorization Server Metadata document. Only the fields a standard
// Authorization Code + PKCE client needs are populated.
type oauthAuthorizationServerMetadata struct {
	Issuer                        string   `json:"issuer"`
	AuthorizationEndpoint         string   `json:"authorization_endpoint"`
	TokenEndpoint                 string   `json:"token_endpoint"`
	ResponseTypesSupported        []string `json:"response_types_supported"`
	GrantTypesSupported           []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
}

// collectOAuthProxySources returns every configured source that implements
// sources.OAuthProxySource and has client-side OAuth enabled, keyed by
// source name.
func collectOAuthProxySources(sourcesMap map[string]sources.Source) map[string]sources.OAuthProxySource {
	matches := make(map[string]sources.OAuthProxySource)
	for name, src := range sourcesMap {
		proxySrc, ok := src.(sources.OAuthProxySource)
		if !ok || !proxySrc.ClientOAuthEnabled() {
			continue
		}
		matches[name] = proxySrc
	}
	return matches
}

// mountOAuthProxyRoutes registers OAuth discovery and token proxy endpoints
// (see sources.OAuthProxySource) on r for every configured source that
// needs them, namespaced by source name so any number of such sources can
// coexist on the same toolbox instance:
//
//	GET  /oauth-proxy/{sourceName}/.well-known/oauth-authorization-server
//	POST /oauth-proxy/{sourceName}/token
//
// It is a no-op — zero added routes, zero behavior change — when no
// configured source needs it.
func mountOAuthProxyRoutes(ctx context.Context, r chi.Router, sourcesMap map[string]sources.Source, caCertFile string, logger log.Logger) error {
	matches := collectOAuthProxySources(sourcesMap)
	if len(matches) == 0 {
		return nil
	}

	httpClient, err := newOAuthProxyHTTPClient(caCertFile)
	if err != nil {
		return fmt.Errorf("failed to configure HTTP client for OAuth proxy: %w", err)
	}

	// Sort names so route registration (and log output) is deterministic.
	names := make([]string, 0, len(matches))
	for name := range matches {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		src := matches[name]

		authEndpoint := src.OAuthAuthorizationEndpoint()
		tokenEndpoint := src.OAuthTokenEndpoint()
		if authEndpoint == "" || tokenEndpoint == "" {
			return fmt.Errorf("oauth proxy: source %q has client-side OAuth enabled but returned an empty authorization or token endpoint", name)
		}

		capabilities := oauthCapabilitiesFor(src)
		base := "/oauth-proxy/" + name

		logger.InfoContext(ctx, fmt.Sprintf(
			"OAuth discovery/token proxy enabled for source %q at %s: proxying authorization endpoint %s "+
				"and token endpoint %s so browser-based MCP clients can complete OAuth without hitting CORS.",
			name, base, authEndpoint, tokenEndpoint))

		r.Get(base+"/.well-known/oauth-authorization-server", func(w http.ResponseWriter, req *http.Request) {
			oauthDiscoveryHandler(w, req, base, authEndpoint, capabilities)
		})

		r.Post(base+"/token", func(w http.ResponseWriter, req *http.Request) {
			oauthTokenProxyHandler(w, req, tokenEndpoint, httpClient, logger)
		})
	}

	return nil
}

// oauthCapabilities holds the OAuth capabilities to advertise for one
// source's proxied discovery document.
type oauthCapabilities struct {
	responseTypesSupported        []string
	grantTypesSupported           []string
	codeChallengeMethodsSupported []string
}

// oauthCapabilitiesFor returns src's advertised OAuth capabilities: the
// defaults, unless src also implements sources.OAuthProxyCapabilities, in
// which case its values are used instead.
func oauthCapabilitiesFor(src sources.OAuthProxySource) oauthCapabilities {
	caps := oauthCapabilities{
		responseTypesSupported:        defaultOAuthResponseTypesSupported,
		grantTypesSupported:           defaultOAuthGrantTypesSupported,
		codeChallengeMethodsSupported: defaultOAuthCodeChallengeMethodsSupported,
	}
	if custom, ok := src.(sources.OAuthProxyCapabilities); ok {
		caps.responseTypesSupported = custom.OAuthResponseTypesSupported()
		caps.grantTypesSupported = custom.OAuthGrantTypesSupported()
		caps.codeChallengeMethodsSupported = custom.OAuthCodeChallengeMethodsSupported()
	}
	return caps
}

// requestBaseURL reconstructs the scheme+host the client used to reach this
// toolbox instance, honoring X-Forwarded-Proto for deployments behind a
// reverse proxy or load balancer that terminates TLS.
func requestBaseURL(req *http.Request) string {
	scheme := "http"
	if proto := req.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if req.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, req.Host)
}

// oauthDiscoveryHandler serves an OAuth Authorization Server Metadata
// document for one source's proxy, mounted at routeBase (e.g.
// "/oauth-proxy/looker"). authorizationEndpoint points directly at the
// authorization server, since the initial redirect there happens via
// top-level browser navigation and is not subject to CORS. token_endpoint
// deliberately points back at THIS toolbox instance's own proxy under
// routeBase (not at the authorization server's real token endpoint
// directly), so that any standard OAuth or MCP client that reads this
// document and then POSTs its token exchange to the advertised
// token_endpoint is routed through the CORS-safe proxy automatically,
// rather than hitting the authorization server's CORS-less token endpoint
// directly and failing.
func oauthDiscoveryHandler(w http.ResponseWriter, req *http.Request, routeBase string, authorizationEndpoint string, capabilities oauthCapabilities) {
	toolboxBaseURL := requestBaseURL(req)
	metadata := oauthAuthorizationServerMetadata{
		Issuer:                        toolboxBaseURL + routeBase,
		AuthorizationEndpoint:         authorizationEndpoint,
		TokenEndpoint:                 toolboxBaseURL + routeBase + "/token",
		ResponseTypesSupported:        capabilities.responseTypesSupported,
		GrantTypesSupported:           capabilities.grantTypesSupported,
		CodeChallengeMethodsSupported: capabilities.codeChallengeMethodsSupported,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(metadata)
}

// oauthTokenProxyHandler forwards a token exchange request to
// tokenEndpoint (the authorization server's real token endpoint) server-side,
// where CORS does not apply, and relays the response back to the caller. It
// accepts both application/x-www-form-urlencoded (the RFC 6749 standard
// content type for token requests) and application/json, re-encoding form
// bodies as JSON since some authorization servers (e.g. Looker) only accept
// JSON on this endpoint.
func oauthTokenProxyHandler(w http.ResponseWriter, req *http.Request, tokenEndpoint string, httpClient *http.Client, logger log.Logger) {
	w.Header().Set("Content-Type", "application/json")

	contentType := req.Header.Get("Content-Type")
	var bodyBytes []byte

	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if err := req.ParseForm(); err != nil {
			logger.ErrorContext(req.Context(), fmt.Sprintf("oauth proxy: failed to parse form body: %v", err))
			http.Error(w, `{"error":"invalid_request","error_description":"failed to parse form body"}`, http.StatusBadRequest)
			return
		}
		payload := map[string]string{}
		for _, key := range []string{"grant_type", "code", "code_verifier", "client_id", "client_secret", "redirect_uri", "refresh_token"} {
			if val := req.FormValue(key); val != "" {
				payload[key] = val
			}
		}
		var err error
		bodyBytes, err = json.Marshal(payload)
		if err != nil {
			logger.ErrorContext(req.Context(), fmt.Sprintf("oauth proxy: failed to marshal token payload as JSON: %v", err))
			http.Error(w, `{"error":"server_error"}`, http.StatusInternalServerError)
			return
		}
		contentType = "application/json"
	} else {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			logger.ErrorContext(req.Context(), fmt.Sprintf("oauth proxy: failed to read token request body: %v", err))
			http.Error(w, `{"error":"server_error"}`, http.StatusInternalServerError)
			return
		}
	}

	logger.DebugContext(req.Context(), fmt.Sprintf(
		"oauth proxy: forwarding token request to %s, content-type=%q, body_len=%d",
		tokenEndpoint, contentType, len(bodyBytes)))

	proxyReq, err := http.NewRequestWithContext(req.Context(), http.MethodPost, tokenEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		logger.ErrorContext(req.Context(), fmt.Sprintf("oauth proxy: failed to build proxy request: %v", err))
		http.Error(w, `{"error":"server_error"}`, http.StatusInternalServerError)
		return
	}
	proxyReq.Header.Set("Content-Type", contentType)

	resp, err := httpClient.Do(proxyReq)
	if err != nil {
		logger.ErrorContext(req.Context(), fmt.Sprintf("oauth proxy: token request to authorization server failed: %v", err))
		http.Error(w, `{"error":"server_error","error_description":"failed to contact authorization server token endpoint"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		logger.ErrorContext(req.Context(), fmt.Sprintf("oauth proxy: failed to copy token response: %v", err))
	}
}

// newOAuthProxyHTTPClient constructs the HTTP client used to reach the
// authorization server's token endpoint from oauthTokenProxyHandler. It is
// shared across all sources' proxies on this toolbox instance.
//
// TLS verification is ALWAYS on. If caCertFile is empty, this returns a
// client using the system's default CA trust store. It only deviates from
// default verification when the operator explicitly opts in with
// --oauth-proxy-ca-cert-file, in which case that CA is *added* to the trust
// pool (never used to skip verification), for deployments where the
// authorization server's certificate is signed by a private/internal CA.
func newOAuthProxyHTTPClient(caCertFile string) (*http.Client, error) {
	transport := &http.Transport{}

	if caCertFile != "" {
		caCert, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read oauth-proxy-ca-cert-file %q: %w", caCertFile, err)
		}

		certPool, err := x509.SystemCertPool()
		if err != nil || certPool == nil {
			certPool = x509.NewCertPool()
		}
		if ok := certPool.AppendCertsFromPEM(caCert); !ok {
			return nil, fmt.Errorf("failed to parse PEM certificate(s) from oauth-proxy-ca-cert-file %q", caCertFile)
		}

		transport.TLSClientConfig = &tls.Config{
			RootCAs: certPool,
		}
	}

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}, nil
}
