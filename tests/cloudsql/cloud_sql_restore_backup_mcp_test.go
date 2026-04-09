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

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"
	sqladmin "google.golang.org/api/sqladmin/v1"

	_ "github.com/googleapis/mcp-toolbox/internal/tools/cloudsql/cloudsqlrestorebackup"
)

var (
	restoreBackupToolKind = "cloud-sql-restore-backup"
)

type restoreBackupTransport struct {
	transport http.RoundTripper
	url       *url.URL
}

func (t *restoreBackupTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.String(), "https://sqladmin.googleapis.com") {
		req.URL.Scheme = t.url.Scheme
		req.URL.Host = t.url.Host
	}
	return t.transport.RoundTrip(req)
}

type masterRestoreBackupHandler struct {
	t *testing.T
}

func (h *masterRestoreBackupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.UserAgent(), "genai-toolbox/") {
		h.t.Errorf("User-Agent header not found")
	}
	var body sqladmin.InstancesRestoreBackupRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.t.Fatalf("failed to decode request body: %v", err)
	} else {
		h.t.Logf("Received request body: %+v", body)
	}

	var expectedBody sqladmin.InstancesRestoreBackupRequest
	var response any
	var statusCode int

	switch {
	case body.Backup != "":
		expectedBody = sqladmin.InstancesRestoreBackupRequest{
			Backup: "projects/p1/backups/test-uid",
		}
		response = map[string]any{"name": "op1", "status": "PENDING"}
		statusCode = http.StatusOK
	case body.BackupdrBackup != "":
		expectedBody = sqladmin.InstancesRestoreBackupRequest{
			BackupdrBackup: "projects/p1/locations/us-central1/backupVaults/test-vault/dataSources/test-ds/backups/test-uid",
		}
		response = map[string]any{"name": "op1", "status": "PENDING"}
		statusCode = http.StatusOK
	case body.RestoreBackupContext != nil:
		expectedBody = sqladmin.InstancesRestoreBackupRequest{
			RestoreBackupContext: &sqladmin.RestoreBackupContext{
				Project:     "p1",
				InstanceId:  "source",
				BackupRunId: 12345,
			},
		}
		response = map[string]any{"name": "op1", "status": "PENDING"}
		statusCode = http.StatusOK
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": `oaraneter "backup_id" is required`,
		})
		return
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

func getRestoreBackupToolsConfig() map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-cloud-sql-source": map[string]any{
				"kind": "cloud-sql-admin",
			},
		},
		"tools": map[string]any{
			"restore-backup": map[string]any{
				"kind":   restoreBackupToolKind,
				"source": "my-cloud-sql-source",
			},
		},
	}
}

func TestRestoreBackupToolMCP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	handler := &masterRestoreBackupHandler{t: t}
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
	http.DefaultClient.Transport = &restoreBackupTransport{
		transport: originalTransport,
		url:       serverURL,
	}
	t.Cleanup(func() {
		http.DefaultClient.Transport = originalTransport
	})

	toolsFile := getRestoreBackupToolsConfig()
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
			name:     "successful restore with standard backup",
			toolName: "restore-backup",
			args:     map[string]any{"target_project": "p1", "target_instance": "instance-standard", "backup_id": "12345", "source_project": "p1", "source_instance": "source"},
			want:     `{"name":"op1","status":"PENDING"}`,
		},
		{
			name:     "successful restore with project level backup",
			toolName: "restore-backup",
			args:     map[string]any{"target_project": "p1", "target_instance": "instance-project-level", "backup_id": "projects/p1/backups/test-uid"},
			want:     `{"name":"op1","status":"PENDING"}`,
		},
		{
			name:     "successful restore with BackupDR backup",
			toolName: "restore-backup",
			args:     map[string]any{"target_project": "p1", "target_instance": "instance-project-level", "backup_id": "projects/p1/locations/us-central1/backupVaults/test-vault/dataSources/test-ds/backups/test-uid"},
			want:     `{"name":"op1","status":"PENDING"}`,
		},
		{
			name:        "missing source instance info for standard backup",
			toolName:    "restore-backup",
			args:        map[string]any{"target_project": "p1", "target_instance": "instance-project-level", "backup_id": "12345"},
			expectError: true,
			wantError:   `source project and instance are required when restoring via backup ID`,
		},
		{
			name:        "missing backup identifier",
			toolName:    "restore-backup",
			args:        map[string]any{"target_project": "p1", "target_instance": "instance-project-level"},
			expectError: true,
			wantError:   `parameter "backup_id" is required`,
		},
		{
			name:        "missing target instance info",
			toolName:    "restore-backup",
			args:        map[string]any{"backup_id": "12345"},
			expectError: true,
			wantError:   `parameter "target_project" is required`,
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
