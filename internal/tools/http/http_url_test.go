// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"net/url"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

func TestGetURLHostOverride(t *testing.T) {
	testCases := []struct {
		name         string
		pathParam    string
		expectError  bool
		expectedHost string
	}{
		{
			name:         "at sign in path is not a host override",
			pathParam:    "@evil.com/v1",
			expectError:  false,
			expectedHost: "api.good.com",
		},
		{
			name:        "absolute url in path is rejected",
			pathParam:   "https://evil.com/v1",
			expectError: true,
		},
		{
			name:        "authority override in path is rejected",
			pathParam:   "//evil.com/v1",
			expectError: true,
		},
	}

	baseURL := "https://api.good.com"
	path := "{{.pathParam}}"
	pathParams := parameters.Parameters{parameters.NewStringParameter("pathParam", "path")}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			paramsMap := map[string]any{"pathParam": tc.pathParam}

			urlString, err := getURL(baseURL, path, pathParams, nil, nil, paramsMap)
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			parsed, err := url.Parse(urlString)
			if err != nil {
				t.Fatalf("failed to parse URL: %v", err)
			}

			if parsed.Host != tc.expectedHost {
				t.Fatalf("expected host to be %q, got %q", tc.expectedHost, parsed.Host)
			}
		})
	}
}
