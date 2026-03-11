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

package dgraph

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	DgraphSourceType = "dgraph"
	DgraphApiKey     = "api-key"
	DgraphUrl        = os.Getenv("DGRAPH_URL")
)

func getDgraphVars(t *testing.T) map[string]any {
	if DgraphUrl == "" {
		t.Fatal("'DGRAPH_URL' not set")
	}
	return map[string]any{
		"type":      DgraphSourceType,
		"dgraphUrl": DgraphUrl,
		"apiKey":    DgraphApiKey,
	}
}

func TestDgraphToolEndpoints(t *testing.T) {
	sourceConfig := getDgraphVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-dgraph-instance": sourceConfig,
		},
		"tools": map[string]any{
			"my-simple-dql-tool": map[string]any{
				"type":        "dgraph-dql",
				"source":      "my-dgraph-instance",
				"description": "Simple tool to test end to end functionality.",
				"statement":   "{result(func: uid(0x0)) {constant: math(1)}}",
				"isQuery":     true,
				"timeout":     "20s",
				"parameters":  []any{},
			},
		},
	}
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

	// Test tool get endpoint
	tcs := []struct {
		name string
		api  string
		want map[string]any
	}{
		{
			name: "get my-simple-tool",
			api:  "http://127.0.0.1:5000/mcp",
			want: map[string]any{
				"my-simple-dql-tool": map[string]any{
					"description":  "Simple tool to test end to end functionality.",
					"parameters":   []any{},
					"authRequired": []any{},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Create JSONRPC request for tools/list
			mcpReq := map[string]any{
				"jsonrpc": "2.0",
				"id":      "test-list",
				"method":  "tools/list",
			}
			reqBytes, _ := json.Marshal(mcpReq)
			
			resp, err := http.Post(tc.api, "application/json", bytes.NewBuffer(reqBytes))
			if err != nil {
				t.Fatalf("error when sending a request: %s", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			result, ok := body["result"].(map[string]interface{})
			if !ok {
				t.Fatalf("unable to find result in response body")
			}
			
			gotTools, ok := result["tools"].([]interface{})
			if !ok {
				t.Fatalf("unable to find tools array in result")
			}

			// Convert tools array into a map for comparison
			got := make(map[string]interface{})
			for _, toolItem := range gotTools {
				toolMap, ok := toolItem.(map[string]interface{})
				if !ok {
					continue
				}
				name, ok := toolMap["name"].(string)
				if !ok {
					continue
				}
				
				// Reconstruct a map similar to tc.want
				toolEntry := map[string]interface{}{
					"description": toolMap["description"],
				}
				
				if inputSchema, ok := toolMap["inputSchema"].(map[string]interface{}); ok {
					if props, ok := inputSchema["properties"].(map[string]interface{}); ok {
						if params, ok := props["arguments"].(map[string]interface{}); ok {
							if propsArray, ok := params["properties"].(map[string]interface{}); ok {
								var paramList []interface{}
								for paramName, paramSchema := range propsArray {
									paramMap := paramSchema.(map[string]interface{})
									paramEntry := map[string]interface{}{
										"name":        paramName,
										"type":        paramMap["type"],
										"description": paramMap["description"],
									}
									// Only properties defined here are needed to check against legacy output
									// Dgraph tool takes no parameters, so this handles empty arguments
									paramList = append(paramList, paramEntry)
								}
								toolEntry["parameters"] = paramList
							}
						}
					}
				}
				// if parameters is nil, set it to empty array to match tc.want
				if toolEntry["parameters"] == nil {
					toolEntry["parameters"] = []interface{}{}
				}
				toolEntry["authRequired"] = []interface{}{}
				got[name] = toolEntry
			}

			// Compare as JSON strings to handle any ordering differences
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tc.want)
			if string(gotJSON) != string(wantJSON) {
				t.Logf("got %v, want %v", string(gotJSON), string(wantJSON))
				t.Fatalf("tools mismatch")
			}
		})
	}

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name        string
		api         string
		requestBody io.Reader
		want        string
	}{
		{
			name:        "invoke my-simple-dql-tool",
			api:         "http://127.0.0.1:5000/mcp",
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			want:        "{\"result\":[{\"constant\":1}]}",
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Post(tc.api, "application/json", tc.requestBody)
			if err != nil {
				t.Fatalf("error when sending a request: %s", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			resultObj, ok := body["result"].(map[string]interface{})
			if !ok {
				t.Fatalf("unable to find result object in response body")
			}
			contentList, ok := resultObj["content"].([]interface{})
			if !ok || len(contentList) == 0 {
				t.Fatalf("unable to find content array in result")
			}
			firstContent, ok := contentList[0].(map[string]interface{})
			if !ok {
				t.Fatalf("content is not an object")
			}
			got, ok := firstContent["text"].(string)
			if !ok {
				t.Fatalf("unable to find text in content")
			}

			if got != tc.want {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}
