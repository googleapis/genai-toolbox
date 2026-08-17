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

package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// InvokeTestCase defines a test case for invoking a tool.
type InvokeTestCase struct {
	Name          string
	Api           string
	RequestHeader map[string]string
	RequestBody   io.Reader
	Want          string
	IsErr         bool
}

// RunRequests iterates over a slice of InvokeTestCase and processes them, reducing code duplication.
func RunRequests(t *testing.T, tcs []InvokeTestCase) {
	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			resp, respBody := RunRequest(t, http.MethodPost, tc.Api, tc.RequestBody, tc.RequestHeader)
			if resp.StatusCode != http.StatusOK {
				if tc.IsErr {
					return
				}
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(respBody))
			}

			var body map[string]interface{}
			err := json.Unmarshal(respBody, &body)
			if err != nil {
				t.Fatalf("error parsing response body: %s, body: %s", err, string(respBody))
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body: %s", string(respBody))
			}

			if !strings.Contains(got, tc.Want) {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.Want)
			}
		})
	}
}
