// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utility_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/testutils"
	_ "github.com/googleapis/genai-toolbox/internal/tools/utility/envvariable"
	"github.com/googleapis/genai-toolbox/tests"
)

func TestUpdateMCPSettings(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	toolName := "my-update-mcp-settings"
	mcpSettings := map[string]interface{}{
		"mcpServers": map[string]interface{}{},
	}
	mcpSettingsData, err := json.Marshal(mcpSettings)
	if err != nil {
		t.Fatalf("failed to marshal mcp settings: %v", err)
	}

	tmpDir := t.TempDir()
	mcpSettingsFile := filepath.Join(tmpDir, "mcp.json")
	if err := os.WriteFile(mcpSettingsFile, mcpSettingsData, 0644); err != nil {
		t.Fatalf("failed to write mcp settings file: %v", err)
	}

	toolsFile := map[string]any{
		"tools": map[string]any{
			toolName: map[string]any{
				"kind":        "update-mcp-settings",
				"description": "Update MCP settings.",
			},
		},
	}

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

	t.Run("success", func(t *testing.T) {
		params := map[string]interface{}{
			"mcpSettingsFile":           mcpSettingsFile,
			"ALLOYDB_POSTGRES_PROJECT":  "my-project",
			"ALLOYDB_POSTGRES_REGION":   "my-region",
			"ALLOYDB_POSTGRES_CLUSTER":  "my-cluster",
			"ALLOYDB_POSTGRES_INSTANCE": "my-instance",
			"ALLOYDB_POSTGRES_DATABASE": "my-db",
			"ALLOYDB_POSTGRES_USER":     "my-user",
			"ALLOYDB_POSTGRES_PASSWORD": "my-password",
		}
		var result struct{ Result string }
		if err := invoke(toolName, params, &result); err != nil {
			t.Fatalf("tool invocation failed: %v", err)
		}

		expectedResult := "[\"Successfully updated MCP settings file\"]"
		if !reflect.DeepEqual(result.Result, expectedResult) {
			t.Errorf("unexpected result: got %q, want %q", result.Result, expectedResult)
		}

		data, err := os.ReadFile(mcpSettingsFile)
		if err != nil {
			t.Fatalf("failed to read mcp settings file: %v", err)
		}

		var updatedMCPSettings map[string]interface{}
		if err := json.Unmarshal(data, &updatedMCPSettings); err != nil {
			t.Fatalf("failed to unmarshal mcp settings file: %v", err)
		}

		mcpServers, ok := updatedMCPSettings["mcpServers"].(map[string]interface{})
		if !ok {
			t.Fatalf("mcpServers not found in updated settings")
		}

		alloydbServer, ok := mcpServers["alloydb"].(map[string]interface{})
		if !ok {
			t.Fatalf("alloydb server not found in updated settings")
		}

		env, ok := alloydbServer["env"].(map[string]interface{})
		if !ok {
			t.Fatalf("env not found in alloydb server settings")
		}

		expectedEnv := map[string]interface{}{
			"ALLOYDB_POSTGRES_PROJECT":  "my-project",
			"ALLOYDB_POSTGRES_REGION":   "my-region",
			"ALLOYDB_POSTGRES_CLUSTER":  "my-cluster",
			"ALLOYDB_POSTGRES_INSTANCE": "my-instance",
			"ALLOYDB_POSTGRES_DATABASE": "my-db",
			"ALLOYDB_POSTGRES_USER":     "my-user",
			"ALLOYDB_POSTGRES_PASSWORD": "my-password",
		}
		if !reflect.DeepEqual(env, expectedEnv) {
			t.Errorf("unexpected env: got %v, want %v", env, expectedEnv)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		params := map[string]interface{}{
			"mcpSettingsFile":           "non-existent-file.json",
			"ALLOYDB_POSTGRES_PROJECT":  "my-project",
			"ALLOYDB_POSTGRES_REGION":   "my-region",
			"ALLOYDB_POSTGRES_CLUSTER":  "my-cluster",
			"ALLOYDB_POSTGRES_INSTANCE": "my-instance",
			"ALLOYDB_POSTGRES_DATABASE": "my-db",
			"ALLOYDB_POSTGRES_USER":     "my-user",
			"ALLOYDB_POSTGRES_PASSWORD": "my-password",
		}
		var result struct{ Result string }
		err := invoke(toolName, params, &result)
		if err == nil {
			t.Fatal("expected an error but got none")
		}
		expectedError := "failed to read mcp settings file"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("unexpected error: got %v, want to contain %v", err, expectedError)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		invalidJSONFile := filepath.Join(tmpDir, "invalid.json")
		if err := os.WriteFile(invalidJSONFile, []byte("{"), 0644); err != nil {
			t.Fatalf("failed to write invalid json file: %v", err)
		}
		params := map[string]interface{}{
			"mcpSettingsFile":           invalidJSONFile,
			"ALLOYDB_POSTGRES_PROJECT":  "my-project",
			"ALLOYDB_POSTGRES_REGION":   "my-region",
			"ALLOYDB_POSTGRES_CLUSTER":  "my-cluster",
			"ALLOYDB_POSTGRES_INSTANCE": "my-instance",
			"ALLOYDB_POSTGRES_DATABASE": "my-db",
			"ALLOYDB_POSTGRES_USER":     "my-user",
			"ALLOYDB_POSTGRES_PASSWORD": "my-password",
		}
		var result struct{ Result string }
		err := invoke(toolName, params, &result)
		if err == nil {
			t.Fatal("expected an error but got none")
		}
		expectedError := "failed to unmarshal mcp settings file"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("unexpected error: got %v, want to contain %v", err, expectedError)
		}
	})

	t.Run("missing mcpSettingsFile parameter", func(t *testing.T) {
		params := map[string]interface{}{
			"ALLOYDB_POSTGRES_PROJECT":  "my-project",
			"ALLOYDB_POSTGRES_REGION":   "my-region",
			"ALLOYDB_POSTGRES_CLUSTER":  "my-cluster",
			"ALLOYDB_POSTGRES_INSTANCE": "my-instance",
			"ALLOYDB_POSTGRES_DATABASE": "my-db",
			"ALLOYDB_POSTGRES_USER":     "my-user",
			"ALLOYDB_POSTGRES_PASSWORD": "my-password",
		}
		var result struct{ Result string }
		err := invoke(toolName, params, &result)
		if err == nil {
			t.Fatal("expected an error but got none")
		}
		expectedError := "parameter \\\"mcpSettingsFile\\\" is required"
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("unexpected error: got %v, want to contain %v", err, expectedError)
		}
	})
}

func invoke(toolName string, params map[string]interface{}, result interface{}) error {
	url := fmt.Sprintf("http://127.0.0.1:5000/api/tool/%s/invoke", toolName)
	body, err := json.Marshal(params)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}
	return json.NewDecoder(resp.Body).Decode(result)
}
