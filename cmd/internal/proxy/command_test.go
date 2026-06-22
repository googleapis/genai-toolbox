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

package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/server/mcp/jsonrpc"
	"golang.org/x/oauth2"
)

type staticTokenFactory struct {
	token *oauth2.Token
}

func (f staticTokenFactory) TokenSource(context.Context, proxyConfig) (oauth2.TokenSource, error) {
	return oauth2.StaticTokenSource(f.token), nil
}

func newTestProxy(t *testing.T, target string) *stdioProxy {
	t.Helper()

	u, err := parseTarget(target)
	if err != nil {
		t.Fatalf("parseTarget() failed: %v", err)
	}

	return &stdioProxy{
		target:      u,
		client:      http.DefaultClient,
		tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		authHeader:  defaultAuthHeader,
		userAgent:   "test-agent",
	}
}

func TestStdioProxyForwardsRequestsWithAuthAndSession(t *testing.T) {
	type upstreamRequest struct {
		auth            string
		accept          string
		contentType     string
		sessionID       string
		protocolVersion string
		userAgent       string
		body            string
	}
	var got []upstreamRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("ReadAll() failed: %v", err)
			return
		}

		got = append(got, upstreamRequest{
			auth:            r.Header.Get("Authorization"),
			accept:          r.Header.Get("Accept"),
			contentType:     r.Header.Get("Content-Type"),
			sessionID:       r.Header.Get("Mcp-Session-Id"),
			protocolVersion: r.Header.Get("MCP-Protocol-Version"),
			userAgent:       r.Header.Get("User-Agent"),
			body:            string(body),
		})

		w.Header().Set("Content-Type", "application/json")
		switch len(got) {
		case 1:
			w.Header().Set("Mcp-Session-Id", "session-1")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-03-26"}}`))
		case 2:
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"tools","result":{"tools":[]}}`))
		default:
			http.Error(w, "unexpected request", http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	p := newTestProxy(t, srv.URL)
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
		`{"jsonrpc":"2.0","id":"tools","method":"tools/list"}`,
	}, "\n") + "\n"

	var out bytes.Buffer
	if err := p.Start(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	wantOut := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-03-26"}}`,
		`{"jsonrpc":"2.0","id":"tools","result":{"tools":[]}}`,
	}, "\n") + "\n"
	if out.String() != wantOut {
		t.Fatalf("unexpected stdout:\nwant: %s\ngot:  %s", wantOut, out.String())
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 upstream requests, got %d", len(got))
	}
	for i, req := range got {
		if req.auth != "Bearer test-token" {
			t.Errorf("request %d Authorization = %q, want Bearer test-token", i+1, req.auth)
		}
		if !strings.Contains(req.accept, "application/json") || !strings.Contains(req.accept, "text/event-stream") {
			t.Errorf("request %d Accept = %q, want JSON and event-stream", i+1, req.accept)
		}
		if req.contentType != "application/json" {
			t.Errorf("request %d Content-Type = %q, want application/json", i+1, req.contentType)
		}
		if req.userAgent != "test-agent" {
			t.Errorf("request %d User-Agent = %q, want test-agent", i+1, req.userAgent)
		}
	}
	if got[0].sessionID != "" {
		t.Errorf("first request Mcp-Session-Id = %q, want empty", got[0].sessionID)
	}
	if got[1].sessionID != "session-1" {
		t.Errorf("second request Mcp-Session-Id = %q, want session-1", got[1].sessionID)
	}
	if got[1].protocolVersion != "2025-03-26" {
		t.Errorf("second request MCP-Protocol-Version = %q, want 2025-03-26", got[1].protocolVersion)
	}
	if got[0].body != strings.Split(input, "\n")[0] {
		t.Errorf("first request body changed:\nwant: %s\ngot:  %s", strings.Split(input, "\n")[0], got[0].body)
	}
	if got[1].body != strings.Split(input, "\n")[1] {
		t.Errorf("second request body changed:\nwant: %s\ngot:  %s", strings.Split(input, "\n")[1], got[1].body)
	}
}

func TestStdioProxySuppressesNotificationOutput(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("ReadAll() failed: %v", err)
			return
		}
		gotBody = string(body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	p := newTestProxy(t, srv.URL)
	input := `{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}` + "\n"

	var out bytes.Buffer
	if err := p.Start(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if out.String() != "" {
		t.Fatalf("stdout = %q, want empty", out.String())
	}
	if gotBody != strings.TrimSpace(input) {
		t.Fatalf("upstream body = %q, want %q", gotBody, strings.TrimSpace(input))
	}
}

func TestStdioProxyWritesJSONRPCErrorForUpstreamHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no credentials", http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := newTestProxy(t, srv.URL)
	input := `{"jsonrpc":"2.0","id":"abc","method":"tools/list"}` + "\n"

	var out bytes.Buffer
	if err := p.Start(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	var got jsonrpc.JSONRPCError
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &got); err != nil {
		t.Fatalf("stdout is not a JSON-RPC error: %v", err)
	}
	if got.Id != "abc" {
		t.Errorf("error id = %v, want abc", got.Id)
	}
	if got.Error.Code != jsonrpc.UNAUTHORIZED {
		t.Errorf("error code = %d, want %d", got.Error.Code, jsonrpc.UNAUTHORIZED)
	}
	if !strings.Contains(got.Error.Message, "401 Unauthorized") {
		t.Errorf("error message = %q, want upstream status", got.Error.Message)
	}
}

func TestStdioProxyWritesSSEResponseMessages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message\n"))
		_, _ = w.Write([]byte(`data: {"jsonrpc":"2.0","id":1,"result":{"ok":true}}` + "\n\n"))
	}))
	defer srv.Close()

	p := newTestProxy(t, srv.URL)
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n"

	var out bytes.Buffer
	if err := p.Start(context.Background(), strings.NewReader(input), &out); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	want := `{"jsonrpc":"2.0","id":1,"result":{"ok":true}}` + "\n"
	if out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}
}

func TestNewCommandRequiresTarget(t *testing.T) {
	opts := CommandOptions{
		In:  strings.NewReader(""),
		Out: io.Discard,
	}
	cmd := newCommand(opts, staticTokenFactory{token: &oauth2.Token{AccessToken: "test-token"}})
	cmd.SetArgs([]string{})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() succeeded, want required target error")
	}
	if !strings.Contains(err.Error(), `required flag(s) "target" not set`) {
		t.Fatalf("Execute() error = %v, want required target error", err)
	}
}
