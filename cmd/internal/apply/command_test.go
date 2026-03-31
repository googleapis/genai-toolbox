// Copyright 2026 Google LLC
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

package apply

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/cmd/internal"
	"github.com/spf13/cobra"
)

func applyCommand(ctx context.Context, args []string) (string, error) {
	parentCmd := &cobra.Command{Use: "toolbox"}

	buf := new(bytes.Buffer)
	opts := internal.NewToolboxOptions(internal.WithIOStreams(buf, buf))
	internal.PersistentFlags(parentCmd, opts)

	cmd := NewCommand(opts)
	parentCmd.AddCommand(cmd)
	parentCmd.SetArgs(args)
	// Inject the context into the Cobra command
	parentCmd.SetContext(ctx)

	err := parentCmd.Execute()
	return buf.String(), err
}

func TestApply(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 200 OK for the initial GET list requests
		if r.Method == http.MethodGet && !strings.Contains(r.URL.Path, "my-source") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]string{"my-source"}) // existing primitive
			return
		}
		// Return 200 OK for the GET specific primitive (for DeepEqual check)
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "my-source") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"url": "old-url"})
			return
		}
		// Return 204 No Content for the PUT/Apply request
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}))
	defer mockServer.Close()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	yamlContent := `
kind: source
name: my-source
url: new-url
`

	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Prepare Command Arguments
	u, _ := url.Parse(mockServer.URL)
	args := []string{
		"apply",
		"--address", u.Hostname(),
		"--port", u.Port(),
		"--config", configPath, // Assuming your flags support pointing to the file/dir
	}

	// context will automatically shutdown in 1 second.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := applyCommand(ctx, args)
	if err != nil {
		t.Fatalf("apply command failed: %v", err)
	}

	if !strings.Contains(output, "starting apply sequence") {
		t.Errorf("expected logs to show start of sequence, got: %s", output)
	}

	if !strings.Contains(output, "Done applying") {
		t.Errorf("expected logs to show completion, got: %s", output)
	}
}

func TestPrimitivesLoadAndManage(t *testing.T) {
	// This server handles different "kinds" based on the URL path
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var resp []string

		// Determine which list to return based on the URL suffix
		switch r.URL.Path {
		case "/admin/source":
			resp = []string{"my-source1", "my-source2"}
		case "/admin/tool":
			resp = []string{"my-tool1", "my-tool2"}
		default:
			resp = []string{}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	u, _ := url.Parse(mockServer.URL)
	address := u.Hostname()
	port, _ := strconv.Atoi(u.Port())

	p := NewPrimitives()
	ctx := context.Background()

	err := p.Load(ctx, address, port)
	if err != nil {
		t.Fatalf("Failed to load primitives: %v", err)
	}

	// CHeck that load is successful
	expected := primitives{
		"source": {
			"my-source1": struct{}{},
			"my-source2": struct{}{},
		},
		"tool": {
			"my-tool1": struct{}{},
			"my-tool2": struct{}{},
		},
		"authservice":    {}, // Empty maps for types with no data
		"embeddingmodel": {},
		"toolset":        {},
		"prompt":         {},
	}
	if !reflect.DeepEqual(p, expected) {
		t.Errorf("Primitives map does not match expected state.\nGot: %+v\nWant: %+v", p, expected)
	}

	// Test Case: Exists (Success)
	if !p.Exists("source", "my-source1") {
		t.Errorf("Expected 's3-bucket' to exist in 'source'")
	}

	// Test Case: Exists (Case Insensitivity)
	if !p.Exists("SOURCE", "my-source2") {
		t.Errorf("Expected Exists to handle case-insensitive 'kind'")
	}

	// Test Case: Does Not Exist
	if p.Exists("source", "non-existent") {
		t.Error("Did not expect 'non-existent' to exist")
	}

	// Test Case: Remove
	p.Remove("source", "my-source1")
	if p.Exists("source", "my-source1") {
		t.Error("Expected 'my-source1' to be removed")
	}

	// Verify other items in the same kind still exist
	if !p.Exists("source", "my-source2") {
		t.Error("Expected 'my-source2' to remain after removing sibling")
	}

	// CHeck that load is successful
	expectedAfterRemove := primitives{
		"source": {
			"my-source2": struct{}{},
		},
		"tool": {
			"my-tool1": struct{}{},
			"my-tool2": struct{}{},
		},
		"authservice":    {},
		"embeddingmodel": {},
		"toolset":        {},
		"prompt":         {},
	}

	if !reflect.DeepEqual(p, expectedAfterRemove) {
		t.Errorf("Primitives map does not match expected state.\nGot: %+v\nWant: %+v", p, expectedAfterRemove)
	}
}

func TestPrimitivesLoadError(t *testing.T) {
	// Setup a server that returns an error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	u, _ := url.Parse(mockServer.URL)
	address := u.Hostname()
	port, _ := strconv.Atoi(u.Port())

	p := NewPrimitives()
	err := p.Load(context.Background(), address, port)

	if err == nil {
		t.Error("Expected an error when server returns 500, but got nil")
	}
}
