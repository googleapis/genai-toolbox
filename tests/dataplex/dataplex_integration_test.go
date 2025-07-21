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

package dataplex

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	DataplexSourceKind            = "dataplex"
	DataplexSearchEntriesToolKind = "dataplex-search-entries"
	DataplexProject               = os.Getenv("DATAPLEX_PROJECT")
)

func getDataplexVars(t *testing.T) map[string]any {
	switch "" {
	case DataplexProject:
		t.Fatal("'DATAPLEX_PROJECT' not set")
	}
	return map[string]any{
		"kind":    DataplexSourceKind,
		"project": DataplexProject,
	}
}

func TestDataplexToolEndpoints(t *testing.T) {
	sourceConfig := getDataplexVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	toolsFile := getDataplexToolsConfig(sourceConfig)

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

	runDataplexSearchEntriesToolGetTest(t)
	runDataplexSearchEntriesToolInvokeTest(t)
}

func getDataplexToolsConfig(sourceConfig map[string]any) map[string]any {
	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-dataplex-instance": sourceConfig,
		},
		"tools": map[string]any{
			"my-search-entries-tool": map[string]any{
				"kind":        DataplexSearchEntriesToolKind,
				"source":      "my-dataplex-instance",
				"description": "Simple tool to test end to end functionality.",
			},
		},
	}

	return toolsFile
}

func runDataplexSearchEntriesToolGetTest(t *testing.T) {
	resp, err := http.Get("http://127.0.0.1:5000/api/tool/my-search-entries-tool/")
	if err != nil {
		t.Fatalf("error making GET request: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected status code 200, got %d", resp.StatusCode)
	}
	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("error decoding response body: %s", err)
	}
	got, ok := body["tools"]
	if !ok {
		t.Fatalf("unable to find 'tools' key in response body")
	}

	toolsMap, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("tools is not a map")
	}
	tool, ok := toolsMap["my-search-entries-tool"].(map[string]interface{})
	if !ok {
		t.Fatalf("tool not found in manifest")
	}
	params, ok := tool["parameters"].([]interface{})
	if !ok {
		t.Fatalf("parameters not found")
	}
	paramNames := []string{}
	for _, param := range params {
		paramMap, ok := param.(map[string]interface{})
		if ok {
			paramNames = append(paramNames, paramMap["name"].(string))
		}
	}
	expected := []string{"name", "pageSize", "pageToken", "orderBy", "query"}
	for _, want := range expected {
		found := false
		for _, got := range paramNames {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected parameter %q not found in tool parameters", want)
		}
	}
}

func runDataplexSearchEntriesToolInvokeTest(t *testing.T) {
	body := []byte(`{"query":"displayname=users parent:test_dataset"}`)
	resp, err := http.Post("http://127.0.0.1:5000/api/tool/my-search-entries-tool/invoke", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("error making POST request: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("response status code is not 200")
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("error parsing response body")
	}
	resultStr, ok := result["result"].(string)
	if !ok {
		t.Fatalf("expected 'result' to be a string, got %T", result["result"])
	}
	var entries []interface{}
	if err := json.Unmarshal([]byte(resultStr), &entries); err != nil {
		t.Fatalf("error unmarshalling result string: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected at least one entry in the result, got 0, entries: %v", entries)
	}
	entry, ok := entries[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected first entry to be a map, got %T", entries[0])
	}
	if _, ok := entry["dataplex_entry"]; !ok {
		t.Fatalf("expected entry to have 'dataplex_entry' field, got %v", entry)
	}
}
