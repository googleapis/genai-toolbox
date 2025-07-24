// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package neo4j

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	Neo4jSourceKind = "neo4j"
	Neo4jDatabase   = os.Getenv("NEO4J_DATABASE")
	Neo4jUri        = os.Getenv("NEO4J_URI")
	Neo4jUser       = os.Getenv("NEO4J_USER")
	Neo4jPass       = os.Getenv("NEO4J_PASS")
)

// getNeo4jVars retrieves necessary Neo4j connection details from environment variables.
// It fails the test if any required variable is not set.
func getNeo4jVars(t *testing.T) map[string]any {
	switch "" {
	case Neo4jDatabase:
		t.Fatal("'NEO4J_DATABASE' not set")
	case Neo4jUri:
		t.Fatal("'NEO4J_URI' not set")
	case Neo4jUser:
		t.Fatal("'NEO4J_USER' not set")
	case Neo4jPass:
		t.Fatal("'NEO4J_PASS' not set")
	}

	return map[string]any{
		"kind":     Neo4jSourceKind,
		"uri":      Neo4jUri,
		"database": Neo4jDatabase,
		"user":     Neo4jUser,
		"password": Neo4jPass,
	}
}

// TestNeo4jToolEndpoints sets up an integration test server and tests the API endpoints
// for various Neo4j tools, including cypher execution and schema retrieval.
func TestNeo4jToolEndpoints(t *testing.T) {
	sourceConfig := getNeo4jVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	// Write config into a file and pass it to the command.
	// This configuration defines the data source and the tools to be tested.
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-neo4j-instance": sourceConfig,
		},
		"tools": map[string]any{
			"my-simple-cypher-tool": map[string]any{
				"kind":        "neo4j-cypher",
				"source":      "my-neo4j-instance",
				"description": "Simple tool to test end to end functionality.",
				"statement":   "RETURN 1 as a;",
			},
			"my-simple-execute-cypher-tool": map[string]any{
				"kind":        "neo4j-execute-cypher",
				"source":      "my-neo4j-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"my-readonly-execute-cypher-tool": map[string]any{
				"kind":        "neo4j-execute-cypher",
				"source":      "my-neo4j-instance",
				"description": "A readonly cypher execution tool.",
				"readOnly":    true,
			},
			"my-schema-tool": map[string]any{
				"kind":        "neo4j-schema",
				"source":      "my-neo4j-instance",
				"description": "A tool to get the Neo4j schema.",
			},
			"my-schema-tool-with-cache": map[string]any{
				"kind":               "neo4j-schema",
				"source":             "my-neo4j-instance",
				"description":        "A schema tool with a custom cache expiration.",
				"cacheExpireMinutes": 10,
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

	// Test tool `GET` endpoints to verify their manifests are correct.
	tcs := []struct {
		name string
		api  string
		want map[string]any
	}{
		{
			name: "get my-simple-cypher-tool",
			api:  "http://127.0.0.1:5000/api/tool/my-simple-cypher-tool/",
			want: map[string]any{
				"my-simple-cypher-tool": map[string]any{
					"description":  "Simple tool to test end to end functionality.",
					"parameters":   []any{},
					"authRequired": []any{},
				},
			},
		},
		{
			name: "get my-simple-execute-cypher-tool",
			api:  "http://127.0.0.1:5000/api/tool/my-simple-execute-cypher-tool/",
			want: map[string]any{
				"my-simple-execute-cypher-tool": map[string]any{
					"description": "Simple tool to test end to end functionality.",
					"parameters": []any{
						map[string]any{
							"name":        "cypher",
							"type":        "string",
							"required":    true,
							"description": "The cypher to execute.",
							"authSources": []any{},
						},
					},
					"authRequired": []any{},
				},
			},
		},
		{
			name: "get my-schema-tool",
			api:  "http://127.0.0.1:5000/api/tool/my-schema-tool/",
			want: map[string]any{
				"my-schema-tool": map[string]any{
					"description":  "A tool to get the Neo4j schema.",
					"parameters":   []any{},
					"authRequired": []any{},
				},
			},
		},
		{
			name: "get my-schema-tool-with-cache",
			api:  "http://127.0.0.1:5000/api/tool/my-schema-tool-with-cache/",
			want: map[string]any{
				"my-schema-tool-with-cache": map[string]any{
					"description":  "A schema tool with a custom cache expiration.",
					"parameters":   []any{},
					"authRequired": []any{},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(tc.api)
			if err != nil {
				t.Fatalf("error when sending a request: %s", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatalf("response status code is not 200")
			}

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["tools"]
			if !ok {
				t.Fatalf("unable to find tools in response body")
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}

	// Test tool `invoke` endpoints to verify their functionality.
	invokeTcs := []struct {
		name               string
		api                string
		requestBody        io.Reader
		want               string
		wantStatus         int
		wantErrorSubstring string
		validateFunc       func(t *testing.T, body string)
	}{
		{
			name:        "invoke my-simple-cypher-tool",
			api:         "http://127.0.0.1:5000/api/tool/my-simple-cypher-tool/invoke",
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			want:        "[{\"a\":1}]",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "invoke my-simple-execute-cypher-tool",
			api:         "http://127.0.0.1:5000/api/tool/my-simple-execute-cypher-tool/invoke",
			requestBody: bytes.NewBuffer([]byte(`{"cypher": "RETURN 1 as a;"}`)),
			want:        "[{\"a\":1}]",
			wantStatus:  http.StatusOK,
		},
		{
			name:               "invoke readonly tool with write query",
			api:                "http://127.0.0.1:5000/api/tool/my-readonly-execute-cypher-tool/invoke",
			requestBody:        bytes.NewBuffer([]byte(`{"cypher": "CREATE (n:TestNode)"}`)),
			wantStatus:         http.StatusBadRequest,
			wantErrorSubstring: "this tool is read-only and cannot execute write queries",
		},
		{
			name:        "invoke my-schema-tool",
			api:         "http://127.0.0.1:5000/api/tool/my-schema-tool/invoke",
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			wantStatus:  http.StatusOK,
			validateFunc: func(t *testing.T, body string) {
				var result map[string]any
				if err := json.Unmarshal([]byte(body), &result); err != nil {
					t.Fatalf("failed to unmarshal schema result: %v", err)
				}
				// Check for the presence of top-level keys in the schema response.
				expectedKeys := []string{"nodeLabels", "relationships", "constraints", "indexes", "databaseInfo", "statistics"}
				for _, key := range expectedKeys {
					if _, ok := result[key]; !ok {
						t.Errorf("expected key %q not found in schema response", key)
					}
				}
			},
		},
		{
			name:        "invoke my-schema-tool-with-cache",
			api:         "http://127.0.0.1:5000/api/tool/my-schema-tool-with-cache/invoke",
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			wantStatus:  http.StatusOK,
			validateFunc: func(t *testing.T, body string) {
				var result map[string]any
				if err := json.Unmarshal([]byte(body), &result); err != nil {
					t.Fatalf("failed to unmarshal schema result: %v", err)
				}
				// Also check the structure of the schema response for the cached tool.
				expectedKeys := []string{"nodeLabels", "relationships", "constraints", "indexes", "databaseInfo", "statistics"}
				for _, key := range expectedKeys {
					if _, ok := result[key]; !ok {
						t.Errorf("expected key %q not found in schema response", key)
					}
				}
			},
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Post(tc.api, "application/json", tc.requestBody)
			if err != nil {
				t.Fatalf("error when sending a request: %s", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.wantStatus {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code: got %d, want %d: %s", resp.StatusCode, tc.wantStatus, string(bodyBytes))
			}

			if tc.wantStatus == http.StatusOK {
				var body map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&body)
				if err != nil {
					t.Fatalf("error parsing response body")
				}
				got, ok := body["result"].(string)
				if !ok {
					t.Fatalf("unable to find result in response body")
				}

				if tc.validateFunc != nil {
					// Use the custom validation function if provided.
					tc.validateFunc(t, got)
				} else if got != tc.want {
					// Otherwise, perform a direct string comparison.
					t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
				}
			} else {
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("failed to read error response body: %s", err)
				}
				bodyString := string(bodyBytes)
				if !strings.Contains(bodyString, tc.wantErrorSubstring) {
					t.Fatalf("response body %q does not contain expected error %q", bodyString, tc.wantErrorSubstring)
				}
			}
		})
	}
}
