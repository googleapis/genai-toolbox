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
	"testing"
	"time"

	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"

	_ "github.com/googleapis/mcp-toolbox/internal/tools/cloudsql/cloudsqllistdatabases"
)

var (
	listDatabasesToolType = "cloud-sql-list-databases"
)

type listDatabasesTransport struct {
	transport http.RoundTripper
	url       *url.URL
}

func (t *listDatabasesTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.String(), "https://sqladmin.googleapis.com") {
		req.URL.Scheme = t.url.Scheme
		req.URL.Host = t.url.Host
	}
	return t.transport.RoundTrip(req)
}

type masterListDatabasesHandler struct {
	t *testing.T
}

func (h *masterListDatabasesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.UserAgent(), "genai-toolbox/") {
		h.t.Errorf("User-Agent header not found")
	}

	response := map[string]any{
		"items": []map[string]any{
			{
				"name":      "db1",
				"charset":   "utf8",
				"collation": "utf8_general_ci",
			},
			{
				"name":      "db2",
				"charset":   "utf8mb4",
				"collation": "utf8mb4_unicode_ci",
			},
		},
	}
	statusCode := http.StatusOK

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getListDatabasesToolsConfig() map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-cloud-sql-source": map[string]any{
				"type": "cloud-sql-admin",
			},
		},
		"tools": map[string]any{
			"list-databases": map[string]any{
				"type":   listDatabasesToolType,
				"source": "my-cloud-sql-source",
			},
		},
	}
}

func TestListDatabasesToolMCP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	handler := &masterListDatabasesHandler{t: t}
	server := httptest.NewServer(handler)
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse server URL: %v", err)
	}

	originalTransport := http.DefaultClient.Transport
	if originalTransport == nil {
		originalTransport = http.DefaultTransport
	}
	http.DefaultClient.Transport = &listDatabasesTransport{
		transport: originalTransport,
		url:       serverURL,
	}
	t.Cleanup(func() {
		http.DefaultClient.Transport = originalTransport
	})

	toolsFile := getListDatabasesToolsConfig()
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
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
		wantError   string
	}{
		{
			name:     "successful databases listing",
			toolName: "list-databases",
			args:     map[string]any{"project": "p1", "instance": "i1"},
			want:     `[{"name":"db1","charset":"utf8","collation":"utf8_general_ci"},{"name":"db2","charset":"utf8mb4","collation":"utf8mb4_unicode_ci"}]`,
		},
		{
			name:        "missing instance",
			toolName:    "list-databases",
			args:        map[string]any{"project": "p1"},
			expectError: true,
			wantError:   `parameter "instance" is required`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, mcpResp, err := tests.InvokeMCPTool(t, tc.toolName, tc.args, nil)
			if err != nil {
				t.Fatalf("native error executing %s: %s", tc.toolName, err)
			}
			if statusCode != http.StatusOK {
				t.Fatalf("expected status 200, got %d", statusCode)
			}

			if tc.expectError {
				tests.AssertMCPError(t, mcpResp, tc.wantError)
				return
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
