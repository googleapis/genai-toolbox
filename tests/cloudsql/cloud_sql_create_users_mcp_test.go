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

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"

	_ "github.com/googleapis/mcp-toolbox/internal/tools/cloudsql/cloudsqlcreateusers"
)

var (
	createUserToolType = "cloud-sql-create-users"
)

type createUsersTransport struct {
	transport http.RoundTripper
	url       *url.URL
}

func (t *createUsersTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.String(), "https://sqladmin.googleapis.com") {
		req.URL.Scheme = t.url.Scheme
		req.URL.Host = t.url.Host
	}
	return t.transport.RoundTrip(req)
}

type userCreateRequest struct {
	Name     string `json:"name"`
	Password string `json:"password,omitempty"`
	Type     string `json:"type,omitempty"`
}

type masterCreateUserHandler struct {
	t *testing.T
}

func (h *masterCreateUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.UserAgent(), "genai-toolbox/") {
		h.t.Errorf("User-Agent header not found")
	}

	var body userCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.t.Fatalf("failed to decode request body: %v", err)
	}

	var expectedBody userCreateRequest
	var response any
	var statusCode int

	switch body.Name {
	case "test-user":
		expectedBody = userCreateRequest{Name: "test-user", Password: "password", Type: "BUILT_IN"}
		response = map[string]any{"name": "op1", "status": "PENDING"}
		statusCode = http.StatusOK
	case "iam-user":
		expectedBody = userCreateRequest{Name: "iam-user", Type: "CLOUD_IAM_USER"}
		response = map[string]any{"name": "op2", "status": "PENDING"}
		statusCode = http.StatusOK
	default:
		http.Error(w, fmt.Sprintf("unhandled user name: %s", body.Name), http.StatusInternalServerError)
		return
	}

	if body.Type == "CLOUD_IAM_USER" {
		expectedBody.Password = ""
	}

	if diff := cmp.Diff(expectedBody, body); diff != "" {
		h.t.Errorf("unexpected request body (-want +got):\n%s", diff)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getCreateUsersToolsConfig() map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-cloud-sql-source": map[string]any{
				"type": "cloud-sql-admin",
			},
		},
		"tools": map[string]any{
			"create-user": map[string]any{
				"type":   createUserToolType,
				"source": "my-cloud-sql-source",
			},
		},
	}
}

func TestCreateUsersToolMCP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	handler := &masterCreateUserHandler{t: t}
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
	http.DefaultClient.Transport = &createUsersTransport{
		transport: originalTransport,
		url:       serverURL,
	}
	t.Cleanup(func() {
		http.DefaultClient.Transport = originalTransport
	})

	toolsFile := getCreateUsersToolsConfig()
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
			name:     "successful built-in user creation",
			toolName: "create-user",
			args:     map[string]any{"project": "p1", "instance": "i1", "name": "test-user", "password": "password", "iamUser": false},
			want:     `{"name":"op1","status":"PENDING"}`,
		},
		{
			name:     "successful iam user creation",
			toolName: "create-user",
			args:     map[string]any{"project": "p1", "instance": "i1", "name": "iam-user", "iamUser": true},
			want:     `{"name":"op2","status":"PENDING"}`,
		},
		{
			name:        "missing password for built-in user",
			toolName:    "create-user",
			args:        map[string]any{"project": "p1", "instance": "i1", "name": "test-user", "iamUser": false},
			expectError: true,
			wantError:   `missing 'password' parameter for non-IAM user`,
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
