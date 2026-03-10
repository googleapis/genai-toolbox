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

package cloudsqlmysql_test

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

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
	"google.golang.org/api/sqladmin/v1"
)

var (
	createInstanceToolType = "cloud-sql-mysql-create-instance"
)

type createInstanceTransport struct {
	transport http.RoundTripper
	url       *url.URL
}

func (t *createInstanceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.String(), "https://sqladmin.googleapis.com") {
		req.URL.Scheme = t.url.Scheme
		req.URL.Host = t.url.Host
	}
	return t.transport.RoundTrip(req)
}

type masterHandler struct {
	t *testing.T
}

func (h *masterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.UserAgent(), "genai-toolbox/") {
		h.t.Errorf("User-Agent header not found")
	}

	var body sqladmin.DatabaseInstance
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.t.Fatalf("failed to decode request body: %v", err)
	}

	instanceName := body.Name
	if instanceName == "" {
		http.Error(w, "missing instance name", http.StatusBadRequest)
		return
	}

	var expectedBody sqladmin.DatabaseInstance
	var response any
	var statusCode int

	switch instanceName {
	case "instance1":
		expectedBody = sqladmin.DatabaseInstance{
			Project:         "p1",
			Name:            "instance1",
			DatabaseVersion: "MYSQL_8_0",
			RootPassword:    "password123",
			Settings: &sqladmin.Settings{
				AvailabilityType: "REGIONAL",
				Edition:          "ENTERPRISE_PLUS",
				Tier:             "db-perf-optimized-N-8",
				DataDiskSizeGb:   250,
				DataDiskType:     "PD_SSD",
			},
		}
		response = map[string]any{"name": "op1", "status": "PENDING"}
		statusCode = http.StatusOK
	case "instance2":
		expectedBody = sqladmin.DatabaseInstance{
			Project:         "p2",
			Name:            "instance2",
			DatabaseVersion: "MYSQL_8_4",
			RootPassword:    "password456",
			Settings: &sqladmin.Settings{
				AvailabilityType: "ZONAL",
				Edition:          "ENTERPRISE_PLUS",
				Tier:             "db-perf-optimized-N-2",
				DataDiskSizeGb:   100,
				DataDiskType:     "PD_SSD",
			},
		}
		response = map[string]any{"name": "op2", "status": "RUNNING"}
		statusCode = http.StatusOK
	default:
		http.Error(w, fmt.Sprintf("unhandled instance name: %s", instanceName), http.StatusInternalServerError)
		return
	}

	if expectedBody.Project != body.Project {
		h.t.Errorf("unexpected project: got %q, want %q", body.Project, expectedBody.Project)
	}
	if expectedBody.Name != body.Name {
		h.t.Errorf("unexpected name: got %q, want %q", body.Name, expectedBody.Name)
	}
	if expectedBody.DatabaseVersion != body.DatabaseVersion {
		h.t.Errorf("unexpected databaseVersion: got %q, want %q", body.DatabaseVersion, expectedBody.DatabaseVersion)
	}
	if expectedBody.RootPassword != body.RootPassword {
		h.t.Errorf("unexpected rootPassword: got %q, want %q", body.RootPassword, expectedBody.RootPassword)
	}
	if diff := cmp.Diff(expectedBody.Settings, body.Settings); diff != "" {
		h.t.Errorf("unexpected request body settings (-want +got):\n%s", diff)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func TestCreateInstanceToolEndpoints(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	handler := &masterHandler{t: t}
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
	http.DefaultClient.Transport = &createInstanceTransport{
		transport: originalTransport,
		url:       serverURL,
	}
	t.Cleanup(func() {
		http.DefaultClient.Transport = originalTransport
	})

	var args []string
	toolsFile := getCreateInstanceToolsConfig()
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
		name        string
		toolName    string
		body        string
		want        string
		expectError bool
		errorStatus int
	}{
		{
			name:     "successful creation - production",
			toolName: "create-instance-prod",
			body:     `{"project": "p1", "name": "instance1", "databaseVersion": "MYSQL_8_0", "rootPassword": "password123", "editionPreset": "Production"}`,
			want:     `{"name":"op1","status":"PENDING"}`,
		},
		{
			name:     "successful creation - development",
			toolName: "create-instance-dev",
			body:     `{"project": "p2", "name": "instance2", "rootPassword": "password456", "editionPreset": "Development"}`,
			want:     `{"name":"op2","status":"RUNNING"}`,
		},
		{
			name:     "missing required parameter",
			toolName: "create-instance-prod",
			body:     `{"name": "instance1"}`,
			want:     `{"error":"parameter \"project\" is required"}`,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mcpReq := map[string]any{
				"jsonrpc": "2.0",
				"id":      "test-1",
				"method":  "tools/call",
				"params": map[string]any{
					"name":      tc.toolName,
					"arguments": json.RawMessage(tc.body),
				},
			}
			mcpBytes, _ := json.Marshal(mcpReq)
			req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:5000/mcp", bytes.NewBuffer(mcpBytes))
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK { // MCP always returns 200 OK
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			var mcpResp struct {
				Error *struct {
					Message string `json:"message"`
				} `json:"error"`
				Result *struct {
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
					IsError bool `json:"isError"`
				} `json:"result"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if tc.expectError {
				if mcpResp.Error != nil {
					// We got an MCP protocol-level error, which is expected
					return
				}
				if mcpResp.Result != nil && mcpResp.Result.IsError {
					// We got a tool-level error, which is also expected
					return
				}
				if mcpResp.Result != nil && len(mcpResp.Result.Content) > 0 {
					text := mcpResp.Result.Content[0].Text
					if strings.Contains(strings.ToLower(text), "error") ||
						strings.Contains(strings.ToLower(text), "invalid") ||
						strings.Contains(strings.ToLower(text), "missing") {
						return
					}
					t.Fatalf("expected an error, but got success result: %s", text)
				}
				t.Fatalf("Expected MCP error, but no error found in response payload")
			}

			if mcpResp.Error != nil {
				// To do: if the test defines an expect error message to match on, we should match it
				// against mcpResp.Error.Message
				return
			}

			if mcpResp.Result != nil && mcpResp.Result.IsError {
				if len(mcpResp.Result.Content) > 0 {
					t.Fatalf("expected success, but got MCP tool error: %s", mcpResp.Result.Content[0].Text)
				}
				t.Fatalf("expected success, but got an empty MCP tool error")
			}

			if mcpResp.Result == nil || len(mcpResp.Result.Content) == 0 {
				t.Fatalf("Expected a result with content, but it was empty")
			}

			var got, want map[string]any
			if err := json.Unmarshal([]byte(mcpResp.Result.Content[0].Text), &got); err != nil {
				t.Fatalf("failed to unmarshal result: %v. Raw text: %s", err, mcpResp.Result.Content[0].Text)
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

func getCreateInstanceToolsConfig() map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-cloud-sql-source": map[string]any{
				"type": "cloud-sql-admin",
			},
		},
		"tools": map[string]any{
			"create-instance-prod": map[string]any{
				"type":   createInstanceToolType,
				"source": "my-cloud-sql-source",
			},
			"create-instance-dev": map[string]any{
				"type":   createInstanceToolType,
				"source": "my-cloud-sql-source",
			},
		},
	}
}
