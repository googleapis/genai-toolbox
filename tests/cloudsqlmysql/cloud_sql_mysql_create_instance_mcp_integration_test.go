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

package cloudsqlmysql_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"
	"google.golang.org/api/sqladmin/v1"
)

const createInstanceToolTypeMCP = "cloud-sql-mysql-create-instance"

type createInstanceTransportMCP struct {
	transport http.RoundTripper
	url       *url.URL
}

func (t *createInstanceTransportMCP) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.String(), "https://sqladmin.googleapis.com") {
		req.URL.Scheme = t.url.Scheme
		req.URL.Host = t.url.Host
	}
	return t.transport.RoundTrip(req)
}

type masterHandlerMCP struct {
	t *testing.T
}

func (h *masterHandlerMCP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func TestCreateInstanceToolEndpointsMCP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	handler := &masterHandlerMCP{t: t}
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
	http.DefaultClient.Transport = &createInstanceTransportMCP{
		transport: originalTransport,
		url:       serverURL,
	}
	t.Cleanup(func() {
		http.DefaultClient.Transport = originalTransport
	})

	toolsFile := getCreateInstanceToolsConfigMCP()
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile)
	if err != nil {
		t.Fatalf("command initialization returned an error: %v", err)
	}
	defer cleanup()

	waitCtx, cancelWait := context.WithTimeout(ctx, 10*time.Second)
	defer cancelWait()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %v", err)
	}

	tcs := []struct {
		name        string
		toolName    string
		body        string
		want        string
		expectError bool
	}{
		{
			name:        "verify successful instance creation with production preset",
			toolName:    "create-instance-prod",
			body:        `{"project": "p1", "name": "instance1", "databaseVersion": "MYSQL_8_0", "rootPassword": "password123", "editionPreset": "Production"}`,
			want:        `{"name":"op1","status":"PENDING"}`,
			expectError: false,
		},
		{
			name:        "verify successful instance creation with development preset",
			toolName:    "create-instance-dev",
			body:        `{"project": "p2", "name": "instance2", "rootPassword": "password456", "editionPreset": "Development"}`,
			want:        `{"name":"op2","status":"RUNNING"}`,
			expectError: false,
		},
		{
			name:        "verify missing required parameter returns schema error",
			toolName:    "create-instance-prod",
			body:        `{"name": "instance1"}`,
			want:        `parameter "project" is required`,
			expectError: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var args map[string]any
			if err := json.Unmarshal([]byte(tc.body), &args); err != nil {
				t.Fatalf("failed to unmarshal body: %v", err)
			}

			statusCode, mcpResp, err := tests.InvokeMCPTool(t, tc.toolName, args, nil)
			if err != nil {
				t.Fatalf("native error executing %s: %v", tc.toolName, err)
			}

			if statusCode != http.StatusOK {
				t.Fatalf("expected status 200, got %d", statusCode)
			}

			if tc.expectError {
				tests.AssertMCPError(t, mcpResp, tc.want)
			} else {
				if mcpResp.Result.IsError {
					t.Fatalf("expected success, got error result: %v", mcpResp.Result)
				}
				gotStr := mcpResp.Result.Content[0].Text
				var got, want map[string]any
				if err := json.Unmarshal([]byte(gotStr), &got); err != nil {
					t.Fatalf("failed to unmarshal result: %v", err)
				}
				if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
					t.Fatalf("failed to unmarshal want: %v", err)
				}
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("unexpected result (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func getCreateInstanceToolsConfigMCP() map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-cloud-sql-source": map[string]any{
				"type": "cloud-sql-admin",
			},
		},
		"tools": map[string]any{
			"create-instance-prod": map[string]any{
				"type":   createInstanceToolTypeMCP,
				"source": "my-cloud-sql-source",
			},
			"create-instance-dev": map[string]any{
				"type":   createInstanceToolTypeMCP,
				"source": "my-cloud-sql-source",
			},
		},
	}
}
