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
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"

	_ "github.com/googleapis/mcp-toolbox/internal/tools/cloudsql/cloudsqllistinstances"
)

type transport struct {
	transport http.RoundTripper
	url       *url.URL
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.String(), "https://sqladmin.googleapis.com") {
		req.URL.Scheme = t.url.Scheme
		req.URL.Host = t.url.Host
	}
	return t.transport.RoundTrip(req)
}

func getListInstanceToolsConfig() map[string]any {
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
			"list-instances": map[string]any{
				"type":   "cloud-sql-list-instances",
				"source": "my-cloud-sql-source",
			},
			"list-instances-fail": map[string]any{
				"type":        "cloud-sql-list-instances",
				"description": "list instances",
				"source":      "my-invalid-cloud-sql-source",
			},
		},
	}
}

func TestListInstanceMCP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.UserAgent(), "genai-toolbox/") {
			t.Errorf("User-Agent header not found")
		}
		if r.URL.Path != "/v1/projects/test-project/instances" {
			http.Error(w, fmt.Sprintf("unexpected path: got %q", r.URL.Path), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"items": [{"name": "test-instance", "instanceType": "CLOUD_SQL_INSTANCE"}]}`)
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse server URL: %v", err)
	}

	originalTransport := http.DefaultClient.Transport
	if originalTransport == nil {
		originalTransport = http.DefaultTransport
	}
	http.DefaultClient.Transport = &transport{
		transport: originalTransport,
		url:       serverURL,
	}
	t.Cleanup(func() {
		http.DefaultClient.Transport = originalTransport
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	toolsFile := getListInstanceToolsConfig()
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile("Server ready to serve"), cmd.Out)
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
			name:     "successful operation",
			toolName: "list-instances",
			args:     map[string]any{"project": "test-project"},
			want:     `[{"name":"test-instance","instanceType":"CLOUD_SQL_INSTANCE"}]`,
		},
		{
			name:        "failed operation",
			toolName:    "list-instances-fail",
			args:        map[string]any{"project": "test-project"},
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

			var gotArr, wantArr []map[string]any
			if err := json.Unmarshal([]byte(got), &gotArr); err != nil {
				t.Fatalf("failed to unmarshal result array: %v\nraw: %s", err, got)
			}
			if err := json.Unmarshal([]byte(tc.want), &wantArr); err != nil {
				t.Fatalf("failed to unmarshal want array: %v", err)
			}

			if !reflect.DeepEqual(gotArr, wantArr) {
				t.Fatalf("unexpected result: got %+v, want %+v", gotArr, wantArr)
			}
		})
	}
}
