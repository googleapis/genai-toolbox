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
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestToolsetEndpoint(t *testing.T) {
	toolsMap, toolsets := setUpResources(t)
	ts, shutdown := setUpServer(t, toolsMap, toolsets)
	defer shutdown()

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
			resp, body, err := runRequest(ts, http.MethodGet, fmt.Sprintf("/toolset/%s", tc.toolsetName), nil)
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
					t.Errorf("%q tool not found in manfiest", name)
				}
			}
		})
	}
}

func TestToolGetEndpoint(t *testing.T) {
	toolsMap, toolsets := setUpResources(t)
	ts, shutdown := setUpServer(t, toolsMap, toolsets)
	defer shutdown()

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
			resp, body, err := runRequest(ts, http.MethodGet, fmt.Sprintf("/tool/%s", tc.toolName), nil)
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
					t.Errorf("%q tool not found in manfiest", name)
				}
			}
		})
	}
}
