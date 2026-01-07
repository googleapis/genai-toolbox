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

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestToolsetEndpoint(t *testing.T) {
	mockTools := []MockTool{tool1, tool2}
	toolsMap, toolsets, _, _ := setUpResources(t, mockTools, nil)
	r, shutdown := setUpServer(t, "api", toolsMap, toolsets, nil, nil)
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	// wantResponse is a struct for checks against test cases
	type wantResponse struct {
		statusCode int
		isErr      bool
		version    string
		tools      []string
	}

	testCases := []struct {
		name        string
		toolsetName string
		want        wantResponse
	}{
		{
			name:        "'default' manifest",
			toolsetName: "",
			want: wantResponse{
				statusCode: http.StatusOK,
				version:    fakeVersionString,
				tools:      []string{tool1.Name, tool2.Name},
			},
		},
		{
			name:        "invalid toolset name",
			toolsetName: "some_imaginary_toolset",
			want: wantResponse{
				statusCode: http.StatusNotFound,
				isErr:      true,
			},
		},
		{
			name:        "single toolset 1",
			toolsetName: "tool1_only",
			want: wantResponse{
				statusCode: http.StatusOK,
				version:    fakeVersionString,
				tools:      []string{tool1.Name},
			},
		},
		{
			name:        "single toolset 2",
			toolsetName: "tool2_only",
			want: wantResponse{
				statusCode: http.StatusOK,
				version:    fakeVersionString,
				tools:      []string{tool2.Name},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body, err := runRequest(ts, http.MethodGet, fmt.Sprintf("/toolset/%s", tc.toolsetName), nil, nil)
			if err != nil {
				t.Fatalf("unexpected error during request: %s", err)
			}

			if contentType := resp.Header.Get("Content-type"); contentType != "application/json" {
				t.Fatalf("unexpected content-type header: want %s, got %s", "application/json", contentType)
			}

			if resp.StatusCode != tc.want.statusCode {
				t.Logf("response body: %s", body)
				t.Fatalf("unexpected status code: want %d, got %d", tc.want.statusCode, resp.StatusCode)
			}
			if tc.want.isErr {
				// skip the rest of the checks if this is an error case
				return
			}
			var m tools.ToolsetManifest
			err = json.Unmarshal(body, &m)
			if err != nil {
				t.Fatalf("unable to parse ToolsetManifest: %s", err)
			}
			// Check the version is correct
			if m.ServerVersion != tc.want.version {
				t.Fatalf("unexpected ServerVersion: want %q, got %q", tc.want.version, m.ServerVersion)
			}
			// validate that the tools in the toolset are correct
			for _, name := range tc.want.tools {
				_, ok := m.ToolsManifest[name]
				if !ok {
					t.Errorf("%q tool not found in manifest", name)
				}
			}
		})
	}
}

func TestToolGetEndpoint(t *testing.T) {
	mockTools := []MockTool{tool1, tool2}
	toolsMap, toolsets, _, _ := setUpResources(t, mockTools, nil)
	r, shutdown := setUpServer(t, "api", toolsMap, toolsets, nil, nil)
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	// wantResponse is a struct for checks against test cases
	type wantResponse struct {
		statusCode int
		isErr      bool
		version    string
		tools      []string
	}

	testCases := []struct {
		name     string
		toolName string
		want     wantResponse
	}{
		{
			name:     "tool1",
			toolName: tool1.Name,
			want: wantResponse{
				statusCode: http.StatusOK,
				version:    fakeVersionString,
				tools:      []string{tool1.Name},
			},
		},
		{
			name:     "tool2",
			toolName: tool2.Name,
			want: wantResponse{
				statusCode: http.StatusOK,
				version:    fakeVersionString,
				tools:      []string{tool2.Name},
			},
		},
		{
			name:     "invalid tool",
			toolName: "some_imaginary_tool",
			want: wantResponse{
				statusCode: http.StatusNotFound,
				isErr:      true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body, err := runRequest(ts, http.MethodGet, fmt.Sprintf("/tool/%s", tc.toolName), nil, nil)
			if err != nil {
				t.Fatalf("unexpected error during request: %s", err)
			}

			if contentType := resp.Header.Get("Content-type"); contentType != "application/json" {
				t.Fatalf("unexpected content-type header: want %s, got %s", "application/json", contentType)
			}

			if resp.StatusCode != tc.want.statusCode {
				t.Logf("response body: %s", body)
				t.Fatalf("unexpected status code: want %d, got %d", tc.want.statusCode, resp.StatusCode)
			}
			if tc.want.isErr {
				// skip the rest of the checks if this is an error case
				return
			}
			var m tools.ToolsetManifest
			err = json.Unmarshal(body, &m)
			if err != nil {
				t.Fatalf("unable to parse ToolsetManifest: %s", err)
			}
			// Check the version is correct
			if m.ServerVersion != tc.want.version {
				t.Fatalf("unexpected ServerVersion: want %q, got %q", tc.want.version, m.ServerVersion)
			}
			// validate that the tools in the toolset are correct
			for _, name := range tc.want.tools {
				_, ok := m.ToolsManifest[name]
				if !ok {
					t.Errorf("%q tool not found in manifest", name)
				}
			}
		})
	}
}

func TestToolInvokeEndpoint(t *testing.T) {
	mockTools := []MockTool{tool1, tool2, tool4, tool5}
	toolsMap, toolsets, _, _ := setUpResources(t, mockTools, nil)
	r, shutdown := setUpServer(t, "api", toolsMap, toolsets, nil, nil)
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	testCases := []struct {
		name        string
		toolName    string
		requestBody io.Reader
		want        string
		isErr       bool
	}{
		{
			name:        "tool1",
			toolName:    tool1.Name,
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			want:        "{result:[no_params]}\n",
			isErr:       false,
		},
		{
			name:        "tool2",
			toolName:    tool2.Name,
			requestBody: bytes.NewBuffer([]byte(`{"param1": 1, "param2": 2}`)),
			want:        "{result:[some_params]}\n",
			isErr:       false,
		},
		{
			name:        "invalid tool",
			toolName:    "some_imaginary_tool",
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			want:        "",
			isErr:       true,
		},
		{
			name:        "tool4",
			toolName:    tool4.Name,
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			want:        "",
			isErr:       true,
		},
		{
			name:        "tool5",
			toolName:    tool5.Name,
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			want:        "",
			isErr:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body, err := runRequest(ts, http.MethodPost, fmt.Sprintf("/tool/%s/invoke", tc.toolName), tc.requestBody, nil)
			if err != nil {
				t.Fatalf("unexpected error during request: %s", err)
			}

			if contentType := resp.Header.Get("Content-type"); contentType != "application/json" {
				t.Fatalf("unexpected content-type header: want %s, got %s", "application/json", contentType)
			}

			if resp.StatusCode != http.StatusOK {
				if tc.isErr == true {
					return
				}
				t.Fatalf("response status code is not 200, got %d, %s", resp.StatusCode, string(body))
			}

			got := string(body)

			// Remove `\` and `"` for string comparison
			got = strings.ReplaceAll(got, "\\", "")
			want := strings.ReplaceAll(tc.want, "\\", "")
			got = strings.ReplaceAll(got, "\"", "")
			want = strings.ReplaceAll(want, "\"", "")

			if got != want {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSourceListEndpoint(t *testing.T) {
	sourceA := &MockSource{Name: "source-a", Kind: "postgres"}
	sourceB := &MockSource{Name: "source-b", Kind: "mysql"}
	sourcesMap := map[string]sources.Source{
		"source-a": sourceA,
		"source-b": sourceB,
	}

	r, shutdown := setUpServerWithResources(t, "api", sourcesMap, nil, nil, nil, nil, nil)
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	resp, body, err := runRequest(ts, http.MethodGet, "/source", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error during request: %s", err)
	}

	if contentType := resp.Header.Get("Content-type"); contentType != "application/json" {
		t.Fatalf("unexpected content-type header: want %s, got %s", "application/json", contentType)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: want %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var m SourceListResponse
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("unable to parse SourceListResponse: %s", err)
	}

	if _, ok := m.Sources["source-a"]; !ok {
		t.Fatalf("source-a not found in response")
	}
	if _, ok := m.Sources["source-b"]; !ok {
		t.Fatalf("source-b not found in response")
	}
}

func TestSourceGetEndpoint(t *testing.T) {
	sourceA := &MockSource{
		Name:     "source-a",
		Kind:     "postgres",
		Host:     "127.0.0.1",
		Password: "secret",
	}
	sourcesMap := map[string]sources.Source{
		"source-a": sourceA,
	}

	r, shutdown := setUpServerWithResources(t, "api", sourcesMap, nil, nil, nil, nil, nil)
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	resp, body, err := runRequest(ts, http.MethodGet, "/source/source-a", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error during request: %s", err)
	}

	if contentType := resp.Header.Get("Content-type"); contentType != "application/json" {
		t.Fatalf("unexpected content-type header: want %s, got %s", "application/json", contentType)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: want %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var m SourceListResponse
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("unable to parse SourceListResponse: %s", err)
	}

	sourceInfo, ok := m.Sources["source-a"]
	if !ok {
		t.Fatalf("source-a not found in response")
	}
	if sourceInfo.Config == nil {
		t.Fatalf("expected config for source-a, got nil")
	}
	if host, ok := sourceInfo.Config["host"]; !ok || host != "127.0.0.1" {
		t.Fatalf("expected host to be %q, got %v", "127.0.0.1", host)
	}
	if password, ok := sourceInfo.Config["password"]; !ok || password != "[REDACTED]" {
		t.Fatalf("expected password to be redacted, got %v", password)
	}

	resp, _, err = runRequest(ts, http.MethodGet, "/source/unknown-source", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error during request: %s", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected status code for missing source: want %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestAuthServiceListEndpoint(t *testing.T) {
	authA := &MockAuthService{Name: "auth-a", Kind: "google"}
	authB := &MockAuthService{Name: "auth-b", Kind: "google"}
	authMap := map[string]auth.AuthService{
		"auth-a": authA,
		"auth-b": authB,
	}
	toolsMap := map[string]tools.Tool{
		"tool-auth-a": MockTool{Name: "tool-auth-a", AuthRequired: []string{"auth-a"}},
	}

	r, shutdown := setUpServerWithResources(t, "api", nil, authMap, toolsMap, nil, nil, nil)
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	resp, body, err := runRequest(ts, http.MethodGet, "/authservice", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error during request: %s", err)
	}

	if contentType := resp.Header.Get("Content-type"); contentType != "application/json" {
		t.Fatalf("unexpected content-type header: want %s, got %s", "application/json", contentType)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: want %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var m AuthServiceListResponse
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("unable to parse AuthServiceListResponse: %s", err)
	}

	authAInfo, ok := m.AuthServices["auth-a"]
	if !ok {
		t.Fatalf("auth-a not found in response")
	}
	if authAInfo.HeaderName != "auth-a_token" {
		t.Fatalf("unexpected headerName: want %q, got %q", "auth-a_token", authAInfo.HeaderName)
	}
	if len(authAInfo.Tools) != 1 || authAInfo.Tools[0] != "tool-auth-a" {
		t.Fatalf("unexpected tools list for auth-a: %v", authAInfo.Tools)
	}

	if _, ok := m.AuthServices["auth-b"]; !ok {
		t.Fatalf("auth-b not found in response")
	}
}

func TestAuthServiceGetEndpoint(t *testing.T) {
	authA := &MockAuthService{Name: "auth-a", Kind: "google"}
	authMap := map[string]auth.AuthService{
		"auth-a": authA,
	}
	toolsMap := map[string]tools.Tool{
		"tool-auth-a": MockTool{Name: "tool-auth-a", AuthRequired: []string{"auth-a"}},
	}

	r, shutdown := setUpServerWithResources(t, "api", nil, authMap, toolsMap, nil, nil, nil)
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	resp, body, err := runRequest(ts, http.MethodGet, "/authservice/auth-a", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error during request: %s", err)
	}

	if contentType := resp.Header.Get("Content-type"); contentType != "application/json" {
		t.Fatalf("unexpected content-type header: want %s, got %s", "application/json", contentType)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: want %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var m AuthServiceListResponse
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("unable to parse AuthServiceListResponse: %s", err)
	}

	authAInfo, ok := m.AuthServices["auth-a"]
	if !ok {
		t.Fatalf("auth-a not found in response")
	}
	if authAInfo.HeaderName != "auth-a_token" {
		t.Fatalf("unexpected headerName: want %q, got %q", "auth-a_token", authAInfo.HeaderName)
	}
	if len(authAInfo.Tools) != 1 || authAInfo.Tools[0] != "tool-auth-a" {
		t.Fatalf("unexpected tools list for auth-a: %v", authAInfo.Tools)
	}

	resp, _, err = runRequest(ts, http.MethodGet, "/authservice/unknown-auth", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error during request: %s", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected status code for missing auth service: want %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}
