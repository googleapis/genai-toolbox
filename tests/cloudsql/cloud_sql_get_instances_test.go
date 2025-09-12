// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloudsql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	getInstancesToolKind = "cloud-sql-get-instance"
)

type getInstancesTransport struct {
	transport http.RoundTripper
	url       *url.URL
}

func (t *getInstancesTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.String(), "https://sqladmin.googleapis.com") {
		req.URL.Scheme = t.url.Scheme
		req.URL.Host = t.url.Host
	}
	return t.transport.RoundTrip(req)
}

type instance struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type handler struct {
	mu        sync.Mutex
	instances map[string]*instance
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !strings.HasPrefix(r.URL.Path, "/v1/projects/") {
		http.Error(w, "unexpected path", http.StatusBadRequest)
		return
	}

	// The format is /v1/projects/{project}/instances/{instance_name}
	// We only care about the instance_name for the test
	parts := regexp.MustCompile("/").Split(r.URL.Path, -1)
	instanceName := parts[len(parts)-1]

	inst, ok := h.instances[instanceName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(inst); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func TestGetInstancesToolEndpoints(t *testing.T) {
	h := &handler{
		instances: map[string]*instance{
			"instance-1": {Name: "instance-1", Kind: "sql#instance"},
		},
	}
	server := httptest.NewServer(h)
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse server URL: %v", err)
	}

	originalTransport := http.DefaultClient.Transport
	if originalTransport == nil {
		originalTransport = http.DefaultTransport
	}
	http.DefaultClient.Transport = &getInstancesTransport{
		transport: originalTransport,
		url:       serverURL,
	}
	t.Cleanup(func() {
		http.DefaultClient.Transport = originalTransport
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	toolsFile := getToolsConfig()
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	tcs := []struct {
		name          string
		toolName      string
		body          string
		want          string
		expectError   bool
		wantSubstring bool
	}{
		{
			name:     "successful get instance",
			toolName: "get-instance-1",
			body:     `{"projectId": "p1", "instanceId": "instance-1"}`,
			want:     `{"name":"instance-1","kind":"sql#instance"}`,
		},
		{
			name:        "failed get instance",
			toolName:    "get-instance-2",
			body:        `{"projectId": "p1", "instanceId": "instance-2"}`,
			expectError: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			api := fmt.Sprintf("http://127.0.0.1:5000/api/tool/%s/invoke", tc.toolName)
			req, err := http.NewRequest(http.MethodPost, api, bytes.NewBufferString(tc.body))
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if tc.expectError {
				if resp.StatusCode == http.StatusOK {
					t.Fatal("expected error but got status 200")
				}
				return
			}

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			var result struct {
				Result any `json:"result"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			var gotBytes []byte
			if s, ok := result.Result.(string); ok {
				gotBytes = []byte(s)
			} else {
				var err error
				gotBytes, err = json.Marshal(result.Result)
				if err != nil {
					t.Fatalf("failed to marshal result: %v", err)
				}
			}

			if tc.wantSubstring {
				if !bytes.Contains(gotBytes, []byte(tc.want)) {
					t.Fatalf("unexpected result: got %q, want substring %q", string(gotBytes), tc.want)
				}
				return
			}

			var got, want map[string]any
			if err := json.Unmarshal(gotBytes, &got); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}
			if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
				t.Fatalf("failed to unmarshal want: %v", err)
			}

			if !reflect.DeepEqual(got, want) {
				t.Fatalf("unexpected result: got %+v, want %+v", got, want)
			}
		})
	}
}

func getToolsConfig() map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-cloud-sql-source": map[string]any{
				"kind": "cloud-sql-admin",
			},
			"my-invalid-cloud-sql-source": map[string]any{
				"kind":           "cloud-sql-admin",
				"useClientOAuth": true,
			},
		},
		"tools": map[string]any{
			"get-instance-1": map[string]any{
				"kind":        getInstancesToolKind,
				"description": "get instance 1",
				"source":      "my-cloud-sql-source",
			},
			"get-instance-2": map[string]any{
				"kind":        getInstancesToolKind,
				"description": "get instance 2",
				"source":      "my-invalid-cloud-sql-source",
			},
		},
	}
}
