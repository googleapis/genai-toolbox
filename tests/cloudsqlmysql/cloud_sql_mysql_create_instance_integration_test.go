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
	createInstanceToolKind = "cloud-sql-mysql-create-instance"
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
			name:        "missing required parameter",
			toolName:    "create-instance-prod",
			body:        `{"name": "instance1"}`,
			expectError: true,
			errorStatus: http.StatusBadRequest,
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

			var got, want map[string]any
			if err := json.Unmarshal([]byte(result.Result), &got); err != nil {
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

func getCreateInstanceToolsConfig() map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-cloud-sql-source": map[string]any{
				"kind": "cloud-sql-admin",
			},
		},
		"tools": map[string]any{
			"create-instance-prod": map[string]any{
				"kind":   createInstanceToolKind,
				"source": "my-cloud-sql-source",
			},
			"create-instance-dev": map[string]any{
				"kind":   createInstanceToolKind,
				"source": "my-cloud-sql-source",
			},
		},
	}
}
