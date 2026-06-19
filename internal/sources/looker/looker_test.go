// Copyright 2025 Google LLC
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

package looker

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	toolboxlog "github.com/googleapis/mcp-toolbox/internal/log"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/looker-open-source/sdk-codegen/go/rtl"
)

func TestParseFromYamlLooker(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			kind: source
			name: my-looker-instance
			type: looker
			base_url: http://example.looker.com/
			client_id: jasdl;k;tjl
			client_secret: sdakl;jgflkasdfkfg
			`,
			want: map[string]sources.SourceConfig{
				"my-looker-instance": Config{
					Name:               "my-looker-instance",
					Type:               SourceType,
					BaseURL:            "http://example.looker.com/",
					ClientId:           "jasdl;k;tjl",
					ClientSecret:       "sdakl;jgflkasdfkfg",
					Timeout:            "600s",
					SslVerification:    true,
					UseClientOAuth:     "false",
					ShowHiddenModels:   true,
					ShowHiddenExplores: true,
					ShowHiddenFields:   true,
					Location:           "us",
					SessionLength:      1200,
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got, _, _, _, _, _, err := server.UnmarshalResourceConfig(context.Background(), testutils.FormatYaml(tc.in))
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
			desc: "extra field",
			in: `
			kind: source
			name: my-looker-instance
			type: looker
			base_url: http://example.looker.com/
			client_id: jasdl;k;tjl
			client_secret: sdakl;jgflkasdfkfg
			schema: test-schema
			`,
			err: "error unmarshaling source: unable to parse source \"my-looker-instance\" as \"looker\": [5:1] unknown field \"schema\"\n   2 | client_id: jasdl;k;tjl\n   3 | client_secret: sdakl;jgflkasdfkfg\n   4 | name: my-looker-instance\n>  5 | schema: test-schema\n       ^\n   6 | type: looker",
		},
		{
			desc: "missing required field",
			in: `
			kind: source
			name: my-looker-instance
			type: looker
			client_id: jasdl;k;tjl
			`,
			err: "error unmarshaling source: unable to parse source \"my-looker-instance\" as \"looker\": Key: 'Config.BaseURL' Error:Field validation for 'BaseURL' failed on the 'required' tag",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, _, _, _, err := server.UnmarshalResourceConfig(context.Background(), testutils.FormatYaml(tc.in))
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			errStr := err.Error()
			if errStr != tc.err {
				t.Fatalf("unexpected error: got %q, want %q", errStr, tc.err)
			}
		})
	}
}

func TestGetLookerSDK_ClientIPPropagation(t *testing.T) {
	// 1. Start a local test server
	serverReceivedHeaders := make(http.Header)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range r.Header {
			serverReceivedHeaders[k] = v
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"id": 123}`)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	// 2. Construct looker Config with UseClientOAuth = "true" pointing to the local test server
	cfg := Config{
		Name:            "test-looker",
		Type:            "looker",
		BaseURL:         ts.URL,
		UseClientOAuth:  "true",
		Timeout:         "5s",
		SslVerification: false,
	}

	// 3. Initialize the source
	ctx := context.Background()
	// Inject a logger so Initialize doesn't fail
	logger, err := toolboxlog.NewStdLogger(io.Discard, io.Discard, "DEBUG")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	ctx = util.WithLogger(ctx, logger)
	ctx = util.WithUserAgent(ctx, "test-agent")

	src, err := cfg.Initialize(ctx, nil)
	if err != nil {
		t.Fatalf("failed to initialize source: %v", err)
	}

	lookerSrc, ok := src.(*Source)
	if !ok {
		t.Fatalf("source is not of type *looker.Source")
	}

	// 4. Inject Client IP into the context
	testIP := "203.0.113.195"
	ctxWithIP := util.WithClientIP(ctx, testIP)

	// 5. Retrieve the Looker SDK using GetLookerSDK
	sdk, err := lookerSrc.GetLookerSDK(ctxWithIP, "mock-token-123")
	if err != nil {
		t.Fatalf("GetLookerSDK failed: %v", err)
	}

	// 6. Retrieve session and request a call using the session client
	authSession, ok := sdk.AuthSession.(*rtl.AuthSession)
	if !ok {
		t.Fatalf("SDK session is not *rtl.AuthSession")
	}

	client := authSession.Client
	req, err := http.NewRequestWithContext(ctxWithIP, "GET", ts.URL+"/api/4.0/user", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// 7. Assert headers are correctly propagated to the test server
	if gotIP := serverReceivedHeaders.Get("X-Forwarded-For"); gotIP != testIP {
		t.Errorf("expected X-Forwarded-For to be %q, got %q", testIP, gotIP)
	}
	if gotIP := serverReceivedHeaders.Get("X-Real-IP"); gotIP != testIP {
		t.Errorf("expected X-Real-IP to be %q, got %q", testIP, gotIP)
	}
	if gotAuth := serverReceivedHeaders.Get("Authorization"); gotAuth != "mock-token-123" {
		t.Errorf("expected Authorization header to be %q, got %q", "mock-token-123", gotAuth)
	}
}

// TestGetLookerSDKProxyTransport verifies that the SDK returned for end-user
// token requests is configured with a transport that honors proxy environment
// variables (Proxy is wired to http.ProxyFromEnvironment). It reaches into the
// unexported transportWithAuthHeader, so it lives in the internal test package.
//
// Behavior is asserted structurally rather than by setting proxy env vars and
// issuing a request: http.ProxyFromEnvironment caches the environment the first
// time it is invoked for the lifetime of the process, which makes env-based
// proxy assertions order-dependent and flaky.
func TestGetLookerSDKProxyTransport(t *testing.T) {
	ctx := context.Background()
	logger, err := toolboxlog.NewStdLogger(io.Discard, io.Discard, "DEBUG")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	ctx = util.WithLogger(ctx, logger)
	ctx = util.WithUserAgent(ctx, "test-agent")

	cfg := Config{
		Name:            "test-looker",
		Type:            SourceType,
		BaseURL:         "https://example.looker.com/",
		UseClientOAuth:  "true",
		Timeout:         "5s",
		SslVerification: true,
	}

	src, err := cfg.Initialize(ctx, nil)
	if err != nil {
		t.Fatalf("failed to initialize source: %v", err)
	}
	lookerSrc, ok := src.(*Source)
	if !ok {
		t.Fatalf("source is not of type *looker.Source")
	}

	sdk, err := lookerSrc.GetLookerSDK(ctx, "mock-token-123")
	if err != nil {
		t.Fatalf("GetLookerSDK failed: %v", err)
	}

	authSession, ok := sdk.AuthSession.(*rtl.AuthSession)
	if !ok {
		t.Fatalf("SDK session is not *rtl.AuthSession")
	}
	wrapped, ok := authSession.Client.Transport.(*transportWithAuthHeader)
	if !ok {
		t.Fatalf("client transport is not *transportWithAuthHeader, got %T", authSession.Client.Transport)
	}
	base, ok := wrapped.Base.(*http.Transport)
	if !ok {
		t.Fatalf("base transport is not *http.Transport, got %T", wrapped.Base)
	}
	if base.Proxy == nil {
		t.Errorf("expected base transport Proxy to be set so proxy env vars are honored, got nil")
	}
	// VerifySsl is true, so TLS verification should remain enabled.
	if base.TLSClientConfig == nil || base.TLSClientConfig.InsecureSkipVerify {
		t.Errorf("expected TLS verification to be enabled when SslVerification is true")
	}
}

// TestInitializeServiceAccountProxyTransport verifies that the SDK built for the
// service-account auth path (UseClientOAuth: "false") is configured with a
// transport that honors proxy environment variables. Initialize validates the
// settings by calling Me(), so a local server stands in for the Looker instance,
// answering the login and current-user requests.
//
// As with the end-user-token test, the proxy is asserted structurally rather
// than via env vars + a live request, because http.ProxyFromEnvironment caches
// the environment process-wide on first use.
func TestInitializeServiceAccountProxyTransport(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/login"):
			if _, err := w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`)); err != nil {
				t.Errorf("failed to write login response: %v", err)
			}
		case strings.HasSuffix(r.URL.Path, "/user"):
			if _, err := w.Write([]byte(`{"first_name":"Test","last_name":"User"}`)); err != nil {
				t.Errorf("failed to write user response: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	ctx := context.Background()
	logger, err := toolboxlog.NewStdLogger(io.Discard, io.Discard, "DEBUG")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	ctx = util.WithLogger(ctx, logger)
	ctx = util.WithUserAgent(ctx, "test-agent")

	cfg := Config{
		Name:            "test-looker",
		Type:            SourceType,
		BaseURL:         ts.URL,
		ClientId:        "test-client-id",
		ClientSecret:    "test-client-secret",
		UseClientOAuth:  "false",
		Timeout:         "5s",
		SslVerification: true,
	}

	src, err := cfg.Initialize(ctx, nil)
	if err != nil {
		t.Fatalf("failed to initialize source: %v", err)
	}
	lookerSrc, ok := src.(*Source)
	if !ok {
		t.Fatalf("source is not of type *looker.Source")
	}
	if lookerSrc.Client == nil {
		t.Fatalf("expected service-account SDK client to be set")
	}

	authSession, ok := lookerSrc.Client.AuthSession.(*rtl.AuthSession)
	if !ok {
		t.Fatalf("SDK session is not *rtl.AuthSession, got %T", lookerSrc.Client.AuthSession)
	}
	// The SDK wraps our transport in oauth2.Transport -> rtl.transportWithHeaders,
	// so walk the exported Base fields down to the underlying *http.Transport.
	base := baseHTTPTransport(authSession.Client.Transport)
	if base == nil {
		t.Fatalf("could not find *http.Transport in client transport chain")
	}
	if base.Proxy == nil {
		t.Errorf("expected base transport Proxy to be set so proxy env vars are honored, got nil")
	}
	// VerifySsl is true, so TLS verification should remain enabled.
	if base.TLSClientConfig == nil || base.TLSClientConfig.InsecureSkipVerify {
		t.Errorf("expected TLS verification to be enabled when SslVerification is true")
	}
}

// baseHTTPTransport walks a chain of RoundTrippers via their exported "Base"
// field and returns the underlying *http.Transport, or nil if none is found.
func baseHTTPTransport(rt http.RoundTripper) *http.Transport {
	for i := 0; i < 10 && rt != nil; i++ {
		if ht, ok := rt.(*http.Transport); ok {
			return ht
		}
		v := reflect.ValueOf(rt)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			return nil
		}
		f := v.FieldByName("Base")
		if !f.IsValid() || !f.CanInterface() {
			return nil
		}
		next, ok := f.Interface().(http.RoundTripper)
		if !ok {
			return nil
		}
		rt = next
	}
	return nil
}
