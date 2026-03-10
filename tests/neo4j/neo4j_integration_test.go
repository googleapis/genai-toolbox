// Copyright 2024 Google LLC
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

package neo4j

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"

	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	Neo4jSourceType = "neo4j"
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
		"type":     Neo4jSourceType,
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
				"type":        "neo4j-cypher",
				"source":      "my-neo4j-instance",
				"description": "Simple tool to test end to end functionality.",
				"statement":   "RETURN 1 as a;",
			},
			"my-simple-execute-cypher-tool": map[string]any{
				"type":        "neo4j-execute-cypher",
				"source":      "my-neo4j-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"my-readonly-execute-cypher-tool": map[string]any{
				"type":        "neo4j-execute-cypher",
				"source":      "my-neo4j-instance",
				"description": "A readonly cypher execution tool.",
				"readOnly":    true,
			},
			"my-schema-tool": map[string]any{
				"type":        "neo4j-schema",
				"source":      "my-neo4j-instance",
				"description": "A tool to get the Neo4j schema.",
			},
			"my-schema-tool-with-cache": map[string]any{
				"type":               "neo4j-schema",
				"source":             "my-neo4j-instance",
				"description":        "A schema tool with a custom cache expiration.",
				"cacheExpireMinutes": 10,
			},
			"my-populated-schema-tool": map[string]any{
				"type":        "neo4j-schema",
				"source":      "my-neo4j-instance",
				"description": "A tool to get the Neo4j schema from a populated DB.",
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

	// Test tool `invoke` endpoints to verify their functionality.
	invokeTcs := []struct {
		name               string
		toolName           string
		api                string
		arguments          any
		want               string
		wantStatus         int
		wantErrorSubstring string
		prepareData        func(t *testing.T)
		validateFunc       func(t *testing.T, body string)
	}{
		{
			name:       "invoke my-simple-cypher-tool",
			toolName:   "my-simple-cypher-tool",
			api:        "http://127.0.0.1:5000/mcp",
			arguments:  map[string]any{},
			want:       "[{\"a\":1}]",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invoke my-simple-execute-cypher-tool",
			toolName:   "my-simple-execute-cypher-tool",
			api:        "http://127.0.0.1:5000/mcp",
			arguments:  map[string]any{"cypher": "RETURN 1 as a;"},
			want:       "[{\"a\":1}]",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invoke my-simple-execute-cypher-tool with dry_run",
			toolName:   "my-simple-execute-cypher-tool",
			api:        "http://127.0.0.1:5000/mcp",
			arguments:  map[string]any{"cypher": "MATCH (n:Test) RETURN n", "dry_run": true},
			wantStatus: http.StatusOK,
			validateFunc: func(t *testing.T, body string) {
				var result []map[string]any
				if err := json.Unmarshal([]byte(body), &result); err != nil {
					t.Fatalf("failed to unmarshal dry_run result: %v", err)
				}
				if len(result) == 0 {
					t.Fatalf("expected a query plan, but got an empty result")
				}

				operatorValue, ok := result[0]["operator"]
				if !ok {
					t.Fatalf("expected key 'Operator' not found in dry_run response: %s", body)
				}

				operatorStr, ok := operatorValue.(string)
				if !ok {
					t.Fatalf("expected 'Operator' to be a string, but got %T", operatorValue)
				}

				if operatorStr != "ProduceResults@neo4j" {
					t.Errorf("unexpected operator: got %q, want %q", operatorStr, "ProduceResults@neo4j")
				}

				childrenCount, ok := result[0]["childrenCount"]
				if !ok {
					t.Fatalf("expected key 'ChildrenCount' not found in dry_run response: %s", body)
				}

				if childrenCount.(float64) != 1 {
					t.Errorf("unexpected children count: got %v, want %d", childrenCount, 1)
				}
			},
		},
		{
			name:               "invoke my-simple-execute-cypher-tool with dry_run and invalid syntax",
			toolName:           "my-simple-execute-cypher-tool",
			api:                "http://127.0.0.1:5000/mcp",
			arguments:          map[string]any{"cypher": "RTN 1", "dry_run": true},
			wantStatus:         http.StatusOK,
			wantErrorSubstring: "unable to execute query",
		},
		{
			name:               "invoke readonly tool with write query",
			toolName:           "my-readonly-execute-cypher-tool",
			api:                "http://127.0.0.1:5000/mcp",
			arguments:          map[string]any{"cypher": "CREATE (n:TestNode)"},
			wantStatus:         http.StatusOK,
			wantErrorSubstring: "this tool is read-only and cannot execute write queries",
		},
		{
			name:               "invoke readonly tool with write query and dry_run",
			toolName:           "my-readonly-execute-cypher-tool",
			api:                "http://127.0.0.1:5000/mcp",
			arguments:          map[string]any{"cypher": "CREATE (n:TestNode)", "dry_run": true},
			wantStatus:         http.StatusOK,
			wantErrorSubstring: "this tool is read-only and cannot execute write queries",
		},
		{
			name:       "invoke my-schema-tool",
			toolName:   "my-schema-tool",
			api:        "http://127.0.0.1:5000/mcp",
			arguments:  map[string]any{},
			wantStatus: http.StatusOK,
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
			name:       "invoke my-schema-tool-with-cache",
			toolName:   "my-schema-tool-with-cache",
			api:        "http://127.0.0.1:5000/mcp",
			arguments:  map[string]any{},
			wantStatus: http.StatusOK,
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
		{
			name:       "invoke my-schema-tool with populated data",
			toolName:   "my-populated-schema-tool",
			api:        "http://127.0.0.1:5000/mcp",
			arguments:  map[string]any{},
			wantStatus: http.StatusOK,
			prepareData: func(t *testing.T) {
				ctx := context.Background()
				driver, err := neo4j.NewDriver(Neo4jUri, neo4j.BasicAuth(Neo4jUser, Neo4jPass, ""))
				if err != nil {
					t.Fatalf("failed to create neo4j driver: %v", err)
				}

				// Helper to execute queries for setup and teardown.
				execute := func(query string) {
					session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: Neo4jDatabase})
					defer session.Close(ctx)
					// Use ExecuteWrite to ensure the query is committed before proceeding.
					_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
						_, err := tx.Run(ctx, query, nil)
						return nil, err
					})

					// Don't fail the test on teardown errors (e.g., entity doesn't exist).
					if err != nil && !strings.Contains(query, "DROP") {
						t.Fatalf("query failed: %s\nerror: %v", query, err)
					}
				}

				// Teardown logic is deferred to ensure it runs even if the test fails.
				// The driver will be closed at the end of this block.
				t.Cleanup(func() {
					execute("DROP CONSTRAINT PersonNameUnique IF EXISTS")
					execute("DROP INDEX MovieTitleIndex IF EXISTS")
					execute("MATCH (n) DETACH DELETE n")
					if err := driver.Close(ctx); err != nil {
						t.Errorf("failed to close driver during cleanup: %v", err)
					}
				})

				// Setup: Create constraints, indexes, and data.
				execute("MERGE (p:Person {name: 'Alice'}) MERGE (m:Movie {title: 'The Matrix'}) MERGE (p)-[:ACTED_IN]->(m)")
				execute("CREATE CONSTRAINT PersonNameUnique IF NOT EXISTS FOR (p:Person) REQUIRE p.name IS UNIQUE")
				execute("CREATE INDEX MovieTitleIndex IF NOT EXISTS FOR (m:Movie) ON (m.title)")
			},
			validateFunc: func(t *testing.T, body string) {
				// Define structs for unmarshaling the detailed schema.
				type Property struct {
					Name  string   `json:"name"`
					Types []string `json:"types"`
				}
				type NodeLabel struct {
					Name       string     `json:"name"`
					Properties []Property `json:"properties"`
				}
				type Relationship struct {
					Type      string `json:"type"`
					StartNode string `json:"startNode"`
					EndNode   string `json:"endNode"`
				}
				type Constraint struct {
					Name       string   `json:"name"`
					Label      string   `json:"label"`
					Properties []string `json:"properties"`
				}
				type Index struct {
					Name       string   `json:"name"`
					Label      string   `json:"label"`
					Properties []string `json:"properties"`
				}
				type Schema struct {
					NodeLabels    []NodeLabel    `json:"nodeLabels"`
					Relationships []Relationship `json:"relationships"`
					Constraints   []Constraint   `json:"constraints"`
					Indexes       []Index        `json:"indexes"`
				}

				var schema Schema
				if err := json.Unmarshal([]byte(body), &schema); err != nil {
					t.Fatalf("failed to unmarshal schema json: %v\nResponse body: %s", err, body)
				}

				// --- Validate Node Labels and Properties ---
				var personLabelFound, movieLabelFound bool
				for _, l := range schema.NodeLabels {
					if l.Name == "Person" {
						personLabelFound = true
						propFound := false
						for _, p := range l.Properties {
							if p.Name == "name" {
								propFound = true
								break
							}
						}
						if !propFound {
							t.Errorf("expected Person label to have 'name' property, but it was not found")
						}
					}
					if l.Name == "Movie" {
						movieLabelFound = true
						propFound := false
						for _, p := range l.Properties {
							if p.Name == "title" {
								propFound = true
								break
							}
						}
						if !propFound {
							t.Errorf("expected Movie label to have 'title' property, but it was not found")
						}
					}
				}
				if !personLabelFound {
					t.Error("expected to find 'Person' in nodeLabels")
				}
				if !movieLabelFound {
					t.Error("expected to find 'Movie' in nodeLabels")
				}

				// --- Validate Relationships ---
				relFound := false
				for _, r := range schema.Relationships {
					if r.Type == "ACTED_IN" && r.StartNode == "Person" && r.EndNode == "Movie" {
						relFound = true
						break
					}
				}
				if !relFound {
					t.Errorf("expected to find relationship '(:Person)-[:ACTED_IN]->(:Movie)', but it was not found")
				}

				// --- Validate Constraints ---
				constraintFound := false
				for _, c := range schema.Constraints {
					if c.Name == "PersonNameUnique" && c.Label == "Person" {
						propFound := false
						for _, p := range c.Properties {
							if p == "name" {
								propFound = true
								break
							}
						}
						if propFound {
							constraintFound = true
							break
						}
					}
				}
				if !constraintFound {
					t.Errorf("expected to find constraint 'PersonNameUnique' on Person(name), but it was not found")
				}

				// --- Validate Indexes ---
				indexFound := false
				for _, i := range schema.Indexes {
					if i.Name == "MovieTitleIndex" && i.Label == "Movie" {
						propFound := false
						for _, p := range i.Properties {
							if p == "title" {
								propFound = true
								break
							}
						}
						if propFound {
							indexFound = true
							break
						}
					}
				}
				if !indexFound {
					t.Errorf("expected to find index 'MovieTitleIndex' on Movie(title), but it was not found")
				}
			},
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare data if a preparation function is provided.
			if tc.prepareData != nil {
				tc.prepareData(t)
			}

			payloadBytes, _ := json.Marshal(map[string]any{
				"jsonrpc": "2.0",
				"id":      "1",
				"method":  "tools/call",
				"params": map[string]any{
					"name":      tc.toolName,
					"arguments": tc.arguments,
				},
			})

			resp, err := http.Post(tc.api, "application/json", bytes.NewBuffer(payloadBytes))
			if err != nil {
				t.Fatalf("error when sending a request: %s", err)
			}
			defer resp.Body.Close()

			if tc.wantStatus == http.StatusOK {
				var body map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&body)
				if err != nil {
					t.Fatalf("error parsing response body")
				}

				// Handle tool errors represented via MCP protocol first
				if tc.wantErrorSubstring != "" {
					errMap, hasErr := body["error"].(map[string]interface{})
					if hasErr {
						errMsg, _ := errMap["message"].(string)
						if !strings.Contains(errMsg, tc.wantErrorSubstring) {
							t.Fatalf("response payload %q does not contain expected error %q", errMsg, tc.wantErrorSubstring)
						}
						return
					} else {
						t.Fatalf("expected error containing %q but no error found in body: %v", tc.wantErrorSubstring, body)
					}
				}

				resultMap, hasResult := body["result"].(map[string]interface{})
				if !hasResult && tc.want != "" {
					t.Fatalf("unable to find result in response body")
				}

				contentList, hasContent := resultMap["content"].([]interface{})
				if !hasContent || len(contentList) == 0 {
					t.Fatalf("unable to find result.content[0] in response body: %v", body)
				}
				contentItem := contentList[0].(map[string]interface{})
				got, ok := contentItem["text"].(string)
				if !ok {
					t.Fatalf("unable to extract text value from result.content[0]")
				}

				if tc.validateFunc != nil {
					// Use the custom validation function if provided.
					tc.validateFunc(t, got)
				} else if got != tc.want {
					// Otherwise, perform a direct string comparison.
					t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
				}
			} else {
				if resp.StatusCode != tc.wantStatus {
					bodyBytes, _ := io.ReadAll(resp.Body)
					t.Fatalf("response status code: got %d, want %d: %s", resp.StatusCode, tc.wantStatus, string(bodyBytes))
				}
			}
		})
	}
}
