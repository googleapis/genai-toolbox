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
	"io"
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
)

var (
	executeSqlManyToolType = "cloud-sql-admin-execute-many"
	sqlManyToolType        = "cloud-sql-admin-sql-many"
)

type executeSqlTransport struct {
	transport http.RoundTripper
	url       *url.URL
}

func (t *executeSqlTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.String(), "https://sqladmin.googleapis.com") {
		req.URL.Scheme = t.url.Scheme
		req.URL.Host = t.url.Host
	}
	return t.transport.RoundTrip(req)
}

type masterExecuteSqlHandler struct {
	t *testing.T
}

func (h *masterExecuteSqlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.UserAgent(), "genai-toolbox/") {
		h.t.Errorf("User-Agent header not found")
	}

	// Verify it's an executeSql request
	if !strings.Contains(r.URL.Path, "/executeSql") {
		h.t.Errorf("unexpected URL path: %s", r.URL.Path)
	}

	// Read request body to verify payload if needed
	bodyBytes, _ := io.ReadAll(r.Body)
	var payload map[string]any
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		h.t.Errorf("failed to unmarshal request body: %v", err)
	}

	// Mock response
	response := map[string]any{
		"results": []map[string]any{
			{
				"columns": []map[string]any{
					{
						"name": "result",
						"type": "STRING",
					},
				},
				"rows": []map[string]any{
					{
						"values": []map[string]any{
							{
								"value": "success",
							},
						},
					},
				},
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

func TestExecuteSqlManyToolEndpoints(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	handler := &masterExecuteSqlHandler{t: t}
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
	http.DefaultClient.Transport = &executeSqlTransport{
		transport: originalTransport,
		url:       serverURL,
	}
	t.Cleanup(func() {
		http.DefaultClient.Transport = originalTransport
	})

	args := []string{"--enable-api"}
	toolsFile := getExecuteSqlToolsConfig()
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
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
		body        string
		want        string
		expectError bool
	}{
		{
			name:     "successful execute-sql-many",
			toolName: "execute-sql-many",
			body:     `{"project": "p1", "instanceId": "i1", "database": "db1", "sql": "SELECT 1"}`,
			want:     `{"results":[{"columns":[{"name":"result","type":"STRING"}],"rows":[{"values":[{"value":"success"}]}]}]}`,
		},
		{
			name:     "successful sql-many",
			toolName: "sql-many",
			body:     `{"project": "p1", "instanceId": "i1", "database": "db1", "user_id": "123"}`,
			want:     `{"results":[{"columns":[{"name":"result","type":"STRING"}],"rows":[{"values":[{"value":"success"}]}]}]}`,
		},
		{
			name:        "missing required param in execute-sql-many",
			toolName:    "execute-sql-many",
			body:        `{"project": "p1", "instanceId": "i1", "database": "db1"}`,
			want:        `parameter "sql" is required`,
			expectError: true,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var args map[string]any
			if err := json.Unmarshal([]byte(tc.body), &args); err != nil {
				t.Fatalf("failed to unmarshal body: %v", err)
			}

			statusCode, mcpResp, err := tests.InvokeMCPTool(t, tc.toolName, args, nil)
			if err != nil {
				t.Fatalf("native error executing %s: %s", tc.toolName, err)
			}

			if statusCode != http.StatusOK {
				t.Fatalf("expected status 200, got %d", statusCode)
			}

			if tc.expectError {
				tests.AssertMCPError(t, mcpResp, tc.want)
				return
			}

			if mcpResp.Result.IsError {
				t.Fatalf("expected success result, got error: %v", mcpResp.Result)
			}

			if len(mcpResp.Result.Content) == 0 {
				t.Fatalf("expected at least one content item, got none")
			}

			gotText := mcpResp.Result.Content[0].Text
			var got, want map[string]any
			if err := json.Unmarshal([]byte(gotText), &got); err != nil {
				t.Fatalf("failed to unmarshal gotText: %v\ngotText was: %s", err, gotText)
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

func getExecuteSqlToolsConfig() map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-cloud-sql-source": map[string]any{
				"type": "cloud-sql-admin",
			},
		},
		"tools": map[string]any{
			"execute-sql-many": map[string]any{
				"type":        executeSqlManyToolType,
				"source":      "my-cloud-sql-source",
				"description": "Use this tool to execute sql statement on a specific instance.",
			},
			"sql-many": map[string]any{
				"type":        sqlManyToolType,
				"source":      "my-cloud-sql-source",
				"description": "Use this tool to get user details from a specific instance.",
				"statement":   "SELECT * FROM users WHERE id = {{.user_id}}",
				"templateParameters": []map[string]any{
					{
						"name":        "user_id",
						"type":        "string",
						"description": "The ID of the user.",
					},
				},
			},
		},
	}
}
