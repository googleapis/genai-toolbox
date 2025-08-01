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

package kuzudb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/tests"
	"github.com/kuzudb/go-kuzu"
)

var (
	database   = "/tmp/example.kuzu"
	toolKind   = "kuzudb-cypher"
	sourceKind = "kuzudb"
)

func getSourceConfig() map[string]any {
	return map[string]any{
		"name":          sourceKind,
		"kind":          sourceKind,
		"database":      database,
		"maxNumThreads": 10,
	}
}
func initKuzuDbConnection() error {
	queries := []string{
		"create node table user(name string primary key, age int64, email string)",
		"create node table city(name string primary key, population int64)",
		"create rel table follows(from user to user, since int64)",
		"create rel table livesin(from user to city)",
		fmt.Sprintf("create (u:user {name:'Alice', age:20, email: %q})", tests.ServiceAccountEmail),
		"create (u:user {name:'Jane', age:30, email: 'janedoe@gmail.com'})",
		"create (u:city {name:'London', population:100})",
		"create (u:city {name:'New York', population:200})",
		"match (u1:user), (u2:user) where u1.name='Alice' and u2.name='Jane' create (u1)-[:follows {since: 2019}]->(u2)",
		"match (u:user), (c:city) where u.name='Alice' and c.name='New York' create (u)-[:livesin]->(c)",
	}
	database, err := kuzu.OpenDatabase(database, kuzu.DefaultSystemConfig())
	if err != nil {
		return err
	}
	conn, err := kuzu.OpenConnection(database)
	if err != nil {
		return err
	}
	for _, q := range queries {
		_, err := conn.Query(q)
		if err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

func TestKuzuDbToolEndpoints(t *testing.T) {
	initKuzuDbConnection()
	defer os.Remove(database)
	defer os.Remove(fmt.Sprintf("%s.lock", database))
	defer os.Remove(fmt.Sprintf("%s.wal", database))

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var args []string

	paramToolStatement, paramToolStatement2, authToolStatement := createParamQueries()
	templateParamToolStmt, templateParamToolStmt2 := createTemplateQueries()
	toolsFile := getToolConfig(paramToolStatement, paramToolStatement2, authToolStatement, templateParamToolStmt, templateParamToolStmt2)
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
	tests.RunToolGetTest(t)
	runToolInvokeTest(t)
	runToolInvokeWithTemplateParameters(t, "user")
}

func createParamQueries() (string, string, string) {
	toolStatement := "match (u:user {name:$name}) return u.age, u.name"
	toolStatement2 := "match (a:user)-[:follows {since:$year}]->(b:user) return a.name, b.name"
	authToolStatement := "match (u:user {name:$email}) return u.age, u.name"
	return toolStatement, toolStatement2, authToolStatement
}
func createTemplateQueries() (string, string) {
	toolStatement := "match (u:{{.tableName}} {name:$name}) return u.age, u.name"
	toolStatement2 := "match (a:{{.tableName}})-[:follows { {{.edgeFilter}} :$year}]->(b:user) return a.name, b.name"
	return toolStatement, toolStatement2
}

func getToolConfig(paramToolStatement, paramToolStatement2, authToolStatement, templateParamToolStmt, templateParamToolStmt2 string) map[string]any {
	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-instance": getSourceConfig(),
		},
		"authServices": map[string]any{
			"my-google-auth": map[string]any{
				"kind":     "google",
				"clientId": tests.ClientId,
			},
		},
		"tools": map[string]any{
			"my-simple-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
				"statement":   "Match (a) return a.name order by a.name;",
			},
			"my-param-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with params.",
				"statement":   paramToolStatement,
				"parameters": []any{
					map[string]any{
						"name":        "name",
						"type":        "string",
						"description": "user name",
					},
				},
			},
			"my-param-tool2": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with params.",
				"statement":   paramToolStatement2,
				"parameters": []any{
					map[string]any{
						"name":        "year",
						"type":        "integer",
						"description": "year since when one user follows the other user",
					},
				},
			},
			"my-fail-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test statement with incorrect syntax.",
				"statement":   "SELEC 1;",
			},
			"my-auth-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test authenticated parameters.",
				// statement to auto-fill authenticated parameter
				"statement": authToolStatement,
				"parameters": []map[string]any{
					{
						"name":        "email",
						"type":        "string",
						"description": "user email",
						"authServices": []map[string]string{
							{
								"name":  "my-google-auth",
								"field": "email",
							},
						},
					},
				},
			},
			"my-auth-required-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test auth required invocation.",
				"statement":   "MATCH (a) return a;",
				"authRequired": []string{
					"my-google-auth",
				},
			},
			"select-fields-templateParams-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with template params.",
				"statement":   templateParamToolStmt,
				"parameters": []any{
					map[string]any{
						"name":        "name",
						"type":        "string",
						"description": "user name",
					},
				},
				"templateParameters": []tools.Parameter{
					tools.NewStringParameter("tableName", "some description"),
				},
			},
			"select-filter-templateParams-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with template param filter.",
				"statement":   templateParamToolStmt2,
				"parameters": []any{
					map[string]any{
						"name":        "year",
						"type":        "integer",
						"description": "year since when one user follows the other user",
					},
				},
				"templateParameters": []tools.Parameter{
					tools.NewStringParameter("tableName", "some description"),
					tools.NewStringParameter("edgeFilter", "some description"),
				},
			},
		},
	}

	return toolsFile
}

func runToolInvokeTest(t *testing.T) {
	idToken := "client-id"
	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-simple-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-simple-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          "[{\"a.name\":\"Alice\"},{\"a.name\":\"Jane\"},{\"a.name\":\"London\"},{\"a.name\":\"New York\"}]",
			isErr:         false,
		},
		{
			name:          "invoke my-param-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-param-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"name": "Alice"}`)),
			want:          "[{\"u.age\":20,\"u.name\":\"Alice\"}]",
			isErr:         false,
		},
		{
			name:          "invoke my-param-tool2",
			api:           "http://127.0.0.1:5000/api/tool/my-param-tool2/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"year": 2019}`)),
			want:          "[{\"a.name\":\"Alice\",\"b.name\":\"Jane\"}]",
			isErr:         false,
		},
		{
			name:          "invoke my-param-tool2 with nil response",
			api:           "http://127.0.0.1:5000/api/tool/my-param-tool2/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"year": 2020}`)),
			want:          "null",
			isErr:         false,
		},
		{
			name:          "Invoke my-param-tool without parameters",
			api:           "http://127.0.0.1:5000/api/tool/my-param-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          "[{\"name\":\"Alice\"}]",
			isErr:         false,
		},
		{
			name:          "Invoke my-auth-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Check response body
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

func runToolInvokeWithTemplateParameters(t *testing.T, tableName string) {

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		ddl           bool
		insert        bool
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke select-fields-templateParams-tool",
			ddl:           true,
			api:           "http://127.0.0.1:5000/api/tool/select-fields-templateParams-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"tableName": "%s", "name": "Alice"}`, tableName))),
			want:          "[{\"u.age\":20,\"u.name\":\"Alice\"}]",
			isErr:         false,
		},
		{
			name:          "invoke select-filter-templateParams-tool",
			insert:        true,
			api:           "http://127.0.0.1:5000/api/tool/select-filter-templateParams-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"tableName": "%s", "edgeFilter": "since", "year":2019}`, tableName))),
			want:          "[{\"a.name\":\"Alice\",\"b.name\":\"Jane\"}]",
			isErr:         false,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {

			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Check response body
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
