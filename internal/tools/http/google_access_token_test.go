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
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/tools"
	"golang.org/x/oauth2"
)

func TestInitializeGoogleAccessTokenIsLazy(t *testing.T) {
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(t.TempDir(), "missing-credentials.json"))

	initialized, err := (Config{
		ConfigBase:            tools.ConfigBase{Name: "google-api", Description: "Call a Google API."},
		Type:                  "http",
		Source:                "my-http-source",
		Path:                  "/resource",
		Method:                tools.HTTPMethod(http.MethodGet),
		SendGoogleAccessToken: true,
	}).Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize returned an error before the tool was invoked: %s", err)
	}
	httpTool, ok := initialized.(Tool)
	if !ok {
		t.Fatalf("Initialize returned %T, want Tool", initialized)
	}
	if httpTool.googleAccessTokenProvider == nil {
		t.Fatal("Google access token provider was not configured")
	}
	if httpTool.googleAccessTokenProvider.tokenSource != nil {
		t.Fatal("Google ADC was resolved during tool initialization")
	}
}

func TestSetGoogleAccessToken(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("unable to create request: %s", err)
	}
	req.Header.Set("Authorization", "Bearer configured-token")

	err = setGoogleAccessToken(
		req,
		&adcTokenProvider{
			ctx:         context.Background(),
			tokenSource: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "adc-token"}),
		},
	)
	if err != nil {
		t.Fatalf("setGoogleAccessToken returned an error: %s", err)
	}
	if got, want := req.Header.Get("Authorization"), "Bearer adc-token"; got != want {
		t.Fatalf("unexpected Authorization header: got %q, want %q", got, want)
	}
}

func TestSetGoogleAccessTokenError(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("unable to create request: %s", err)
	}

	err = setGoogleAccessToken(
		req,
		&adcTokenProvider{
			ctx:         context.Background(),
			tokenSource: errorTokenSource{err: errors.New("credentials unavailable")},
		},
	)
	if err == nil {
		t.Fatal("expected setGoogleAccessToken to return an error")
	}
	if !strings.Contains(err.Error(), "credentials unavailable") {
		t.Fatalf("unexpected error: %s", err)
	}
	if got := req.Header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization header was set after token retrieval failed: %q", got)
	}
}

type errorTokenSource struct {
	err error
}

func (s errorTokenSource) Token() (*oauth2.Token, error) {
	return nil, s.err
}
