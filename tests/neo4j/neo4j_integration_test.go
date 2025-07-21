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
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
	"github.com/joho/godotenv"
)

var (
	Neo4jSourceKind = "neo4j"
	Neo4jDatabase   string
	Neo4jUri        string
	Neo4jUser       string
	Neo4jPass       string
)

func init() {
	// Load environment variables from .env file if it exists,
	// this simplifies local testing without needing to set environment variables manually
	_ = godotenv.Load()

	Neo4jDatabase = os.Getenv("NEO4J_DATABASE")
	Neo4jUri = os.Getenv("NEO4J_URI")
	Neo4jUser = os.Getenv("NEO4J_USER")
	Neo4jPass = os.Getenv("NEO4J_PASS")
}

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

func TestNeo4jToolEndpoints(t *testing.T) {
	sourceConfig := getNeo4jVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	// Write config into a file and pass it to command
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
		},
	}
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, waitCancel := context.WithTimeout(ctx, 10*time.Second)
	defer waitCancel()
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
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var resp *http.Response
			resp, err = http.Get(tc.api)
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

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name        string
		api         string
		requestBody io.Reader
		want        string
	}{
		{
			name:        "invoke my-simple-cypher-tool",
			api:         "http://127.0.0.1:5000/api/tool/my-simple-cypher-tool/invoke",
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			want:        "[{\"a\":1}]",
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			var resp *http.Response
			resp, err = http.Post(tc.api, "application/json", tc.requestBody)
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
			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if got != tc.want {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}
