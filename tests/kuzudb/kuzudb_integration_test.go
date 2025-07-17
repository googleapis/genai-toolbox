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
	"github.com/googleapis/genai-toolbox/tests"
	"github.com/kuzudb/go-kuzu"
)

var (
	database = "/tmp/example.kuzu"
	toolKind = "kuzudb-cypher"
)

func getSourceConfig() map[string]any {
	return map[string]any{
		"name":     "kuzudb",
		"kind":     "kuzudb",
		"database": database,
		// "configuration": map[string]any{
		// 	"maxnumthreads": 10,
		// },
	}
}
func initKuzuDbConnection() error {
	queries := []string{
		"create node table user(name string primary key, age int64)",
		"create node table city(name string primary key, population int64)",
		"create rel table follows(from user to user, since int64)",
		"create rel table livesin(from user to city)",
		"create (u:user {name:'Alice', age:20})",
		"create (u:user {name:'Jane', age:30})",
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

	paramToolStatement, paramToolStatement2 := createParamQueries()
	toolsFile := getToolConfig(paramToolStatement, paramToolStatement2)
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
}

func createParamQueries() (string, string) {
	toolStatement := "match (u:user {name:$name}) return u.*"
	toolStatement2 := "match (a:user)-[:follows {since:$year}]->(b:user) return a.name, b.name"
	return toolStatement, toolStatement2
}

func createTemplateQueries() (string, string) {
	toolStatement := "match (u:{{.tableName}}} {name:$name}) return a.*"
	toolStatement2 := "match (a:{{.tableName}})-[:{{.relTableName}} {since:$year}]->(b:{{.tableName}}) return a.name, b.name"
	return toolStatement, toolStatement2
}

func getToolConfig(paramToolStatement, paramToolStatement2 string) map[string]any {
	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-instance": getSourceConfig(),
		},
		// "authServices": map[string]any{
		// 	"my-google-auth": map[string]any{
		// 		"kind":     "google",
		// 		"clientId": tests.ClientId,
		// 	},
		// },
		"tools": map[string]any{
			"my-simple-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
				"statement":   "Match (a) return a.name;",
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
		},
	}

	return toolsFile
}

func runToolInvokeTest(t *testing.T) {
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
			want:          "[{\"a.name\":\"London\"},{\"a.name\":\"New York\"},{\"a.name\":\"Alice\"},{\"a.name\":\"Jane\"}]",
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
