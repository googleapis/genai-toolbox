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
	"testing"
	"time"

	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"
)

var (
	executeSqlManyToolType = "postgres-execute-sql-many"
	sqlManyToolType        = "postgres-sql-many"
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
		errorStatus int
	}{
		{
			name:     "successful execute-sql-many",
			toolName: "execute-sql-many",
			body:     `{"project": "p1", "instance": "i1", "database": "db1", "sql": "SELECT 1"}`,
			want:     `{"results":[{"columns":[{"name":"result","type":"STRING"}],"rows":[{"values":[{"value":"success"}]}]}]}`,
		},
		{
			name:     "successful sql-many",
			toolName: "sql-many",
			body:     `{"project": "p1", "instance": "i1", "database": "db1", "user_id": "123"}`,
			want:     `{"results":[{"columns":[{"name":"result","type":"STRING"}],"rows":[{"values":[{"value":"success"}]}]}]}`,
		},
		{
			name:     "missing required param in execute-sql-many",
			toolName: "execute-sql-many",
			body:     `{"project": "p1", "instance": "i1", "database": "db1"}`,
			want:     `{"error":"parameter \"sql\" is required"}`,
		},
	}

	for _, tc := range tcs {
		tc := tc
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
				if resp.StatusCode != tc.errorStatus {
					bodyBytes, _ := io.ReadAll(resp.Body)
					t.Fatalf("expected status %d but got %d: %s", tc.errorStatus, resp.StatusCode, string(bodyBytes))
				}
				return
			}

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			var result struct {
				Result string `json:"result"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if strings.Contains(result.Result, `"error":`) {
				var gotMap, wantMap map[string]any
				if err := json.Unmarshal([]byte(result.Result), &gotMap); err != nil {
					t.Fatalf("failed to unmarshal result error object: %v", err)
				}
				if err := json.Unmarshal([]byte(tc.want), &wantMap); err != nil {
					t.Fatalf("failed to unmarshal want error object: %v", err)
				}
				if !reflect.DeepEqual(gotMap, wantMap) {
					t.Fatalf("unexpected error result: got %+v, want %+v", gotMap, wantMap)
				}
				return
			}

			var got, want map[string]any
			if err := json.Unmarshal([]byte(result.Result), &got); err != nil {
				t.Fatalf("failed to unmarshal result object: %v. Result was: %s", err, result.Result)
			}
			if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
				t.Fatalf("failed to unmarshal want object: %v", err)
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
