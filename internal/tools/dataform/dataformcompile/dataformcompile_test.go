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

package dataformcompile

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/cmd"
)

func TestDataformCompileTool(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "dataform-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	projectDir := filepath.Join(tmpDir, "dataform-project")
	initCmd := exec.Command("dataform", "init", projectDir, "gcp-project-id", "us-central1")
	if err := initCmd.Run(); err != nil {
		t.Fatalf("dataform init failed: %v", err)
	}

	toolsYAML := `
tools:
  dataform-compile-test:
    kind: "dataform-compile"
    description: "Compiles a dataform project."
`
	toolsFile := filepath.Join(tmpDir, "tools.yaml")
	if err := os.WriteFile(toolsFile, []byte(toolsYAML), 0644); err != nil {
		t.Fatalf("failed to write tools.yaml: %v", err)
	}

	opts := []cmd.Option{
		cmd.WithArgs("--tools-file", toolsFile),
	}
	command := cmd.NewCommand(opts...)

	go func() {
		if err := command.Execute(); err != nil {
			if !strings.Contains(err.Error(), "http: Server closed") {
				t.Errorf("error executing command: %v", err)
			}
		}
	}()

	// Give the server a moment to start up.
	time.Sleep(2 * time.Second)

	requestBody := fmt.Sprintf(`{
		"project_dir": "%s"
	}`, projectDir)

	req, err := http.NewRequest("POST", "http://127.0.0.1:5000/api/tool/dataform-compile-test/invoke", bytes.NewBufferString(requestBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status OK; got %v", resp.Status)
	}

	// Verify that the dataform project was compiled.
	// A simple check for the presence of the compiled json is sufficient.
	if _, err := os.Stat(filepath.Join(projectDir, "target/graph.json")); os.IsNotExist(err) {
		t.Errorf("graph.json not found in project directory, dataform compile likely failed")
	}
}
