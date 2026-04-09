// Copyright 2026 Google LLC
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
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"

	_ "github.com/googleapis/mcp-toolbox/internal/tools/cloudsql/cloudsqlgetinstances"
)

var (
	getInstancesToolType = "cloud-sql-get-instance"
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
	t         *testing.T
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !strings.Contains(r.UserAgent(), "genai-toolbox/") {
		h.t.Errorf("User-Agent header not found")
	}

	if !strings.HasPrefix(r.URL.Path, "/v1/projects/") {
		http.Error(w, "unexpected path", http.StatusBadRequest)
		return
	}

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

func getToolsConfig() map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-cloud-sql-source": map[string]any{
				"type": "cloud-sql-admin",
			},
			"my-invalid-cloud-sql-source": map[string]any{
				"type":           "cloud-sql-admin",
				"useClientOAuth": true,
			},
		},
		"tools": map[string]any{
			"get-instance-1": map[string]any{
				"type":        getInstancesToolType,
				"description": "get instance 1",
				"source":      "my-cloud-sql-source",
			},
			"get-instance-2": map[string]any{
				"type":        getInstancesToolType,
				"description": "get instance 2",
				"source":      "my-invalid-cloud-sql-source",
			},
		},
	}
}

func TestGetInstancesToolMCP(t *testing.T) {
	h := &handler{
		instances: map[string]*instance{
			"instance-1": {Name: "instance-1", Kind: "sql#instance"},
		},
		t: t,
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

	toolsFile := getToolsConfig()
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile)
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
		name        string
		toolName    string
		args        map[string]any
		want        string
		expectError bool
	}{
		{
			name:     "successful get instance",
			toolName: "get-instance-1",
			args:     map[string]any{"projectId": "p1", "instanceId": "instance-1"},
			want:     `{"name":"instance-1","kind":"sql#instance"}`,
		},
		{
			name:        "failed get instance",
			toolName:    "get-instance-2",
			args:        map[string]any{"projectId": "p1", "instanceId": "instance-2"},
			expectError: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, mcpResp, err := tests.InvokeMCPTool(t, tc.toolName, tc.args, nil)
			if err != nil {
				t.Fatalf("native error executing %s: %s", tc.toolName, err)
			}

			if tc.expectError {
				if statusCode != http.StatusOK {
					// Expected failure at HTTP level (e.g. 401)
					return
				}
				if mcpResp.Error != nil || mcpResp.Result.IsError {
					// Expected failure at MCP level
					return
				}
				t.Fatal("expected error result but got success")
				return
			}

			if statusCode != http.StatusOK {
				t.Fatalf("expected status 200, got %d", statusCode)
			}

			if mcpResp.Result.IsError {
				t.Fatalf("%s returned error result: %v", tc.toolName, mcpResp.Result)
			}

			if len(mcpResp.Result.Content) == 0 {
				t.Fatalf("%s returned empty content field", tc.toolName)
			}

			// Gather all the text blocks
			var blocks []string
			for _, content := range mcpResp.Result.Content {
				if content.Type == "text" {
					blocks = append(blocks, strings.TrimSpace(content.Text))
				}
			}

			got := strings.Join(blocks, "")

			var gotMap, wantMap map[string]any
			if err := json.Unmarshal([]byte(got), &gotMap); err != nil {
				t.Fatalf("failed to unmarshal result: %v\nraw: %s", err, got)
			}
			if err := json.Unmarshal([]byte(tc.want), &wantMap); err != nil {
				t.Fatalf("failed to unmarshal want: %v", err)
			}

			if !reflect.DeepEqual(gotMap, wantMap) {
				t.Fatalf("unexpected result: got %+v, want %+v", gotMap, wantMap)
			}
		})
	}
}
