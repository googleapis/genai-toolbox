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

package elasticsearch

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"

	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	ElasticsearchSourceKind = "elasticsearch"
	ElasticsearchToolKind   = "elasticsearch-esql"
	EsAddress               = os.Getenv("ELASTICSEARCH_HOST")
	EsUser                  = os.Getenv("ELASTICSEARCH_USER")
	EsPass                  = os.Getenv("ELASTICSEARCH_PASS")
)

func getElasticsearchVars(t *testing.T) map[string]any {
	if EsAddress == "" {
		t.Fatal("'ELASTICSEARCH_HOST' not set")
	}
	return map[string]any{
		"kind":      ElasticsearchSourceKind,
		"addresses": []string{EsAddress},
		"username":  EsUser,
		"password":  EsPass,
	}
}

func TestElasticsearchToolEndpoints(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	sourceConfig := getElasticsearchVars(t)

	index := "test-index"

	paramToolStatement, idParamToolStatement, nameParamToolStatement, arrayParamToolStatement, authToolStatement := getElasticsearchQueries(index)

	toolsConfig := getElasticsearchToolsConfig(sourceConfig, ElasticsearchToolKind, paramToolStatement, idParamToolStatement, nameParamToolStatement, arrayParamToolStatement, authToolStatement)

	cmd, cleanup, err := tests.StartCmd(ctx, toolsConfig, args...)
	if err != nil {
		t.Fatalf("failed to start cmd: %v", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	esClient, err := elasticsearch.NewBaseClient(elasticsearch.Config{
		Addresses: []string{EsAddress},
		Username:  EsUser,
		Password:  EsPass,
	})
	if err != nil {
		t.Fatalf("error creating the Elasticsearch client: %s", err)
	}

	// Delete index if already exists
	defer func() {
		_, err = esapi.IndicesDeleteRequest{
			Index: []string{index},
		}.Do(ctx, esClient)
		if err != nil {
			t.Fatalf("error deleting index: %s", err)
		}
	}()

	alice := fmt.Sprintf(`{
									"id": 1,
									"name": "Alice",
									"email": "%s"
								}`, tests.ServiceAccountEmail)

	// Index sample documents
	sampleDocs := []string{
		alice,
		`{"id": 2, "name": "Jane", "email": "janedoe@gmail.com"}`,
		`{"id": 3, "name": "Sid"}`,
		`{"id": 4, "name": "null"}`,
	}
	for _, doc := range sampleDocs {
		res, err := esapi.IndexRequest{
			Index:   "test-index",
			Body:    strings.NewReader(doc),
			Refresh: "true",
		}.Do(ctx, esClient)
		if res.IsError() {
			t.Fatalf("error indexing document: %s", res.String())
		}
		if err != nil {
			t.Fatalf("error indexing document: %s", err)
		}
	}

	// Get configs for tests
	select1Want, myToolId3NameAliceWant, myToolById4Want, nullWant, mcpMyToolId3NameAliceWant := getElasticsearchWants()

	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want,
		tests.DisableArrayTest(),

		tests.WithMyToolId3NameAliceWant(myToolId3NameAliceWant),
		tests.WithMyToolById4Want(myToolById4Want),
		tests.WithNullWant(nullWant),
	)
	tests.RunMCPToolCallMethod(t, "", "", tests.WithMcpMyToolId3NameAliceWant(mcpMyToolId3NameAliceWant))
}

func getElasticsearchQueries(index string) (string, string, string, string, string) {
	paramToolStatement := fmt.Sprintf(`FROM %s | WHERE id == ?id OR name == ?name | SORT id ASC`, index)
	idParamToolStatement := fmt.Sprintf(`FROM %s | WHERE id == ?id`, index)
	nameParamToolStatement := fmt.Sprintf(`FROM %s | WHERE name == ?name`, index)
	arrayParamToolStatement := fmt.Sprintf(`FROM %s | WHERE first_name == ?first_name_array`, index) // Not supported yet.
	authToolStatement := fmt.Sprintf(`FROM %s | WHERE name == ?name`, index)
	return paramToolStatement, idParamToolStatement, nameParamToolStatement, arrayParamToolStatement, authToolStatement
}

func getElasticsearchWants() (string, string, string, string, string) {
	select1Want := `[["test@elastic.co","test@elastic.co",1,"Alice","Alice"],["janedoe@gmail.com","janedoe@gmail.com",2,"Jane","Jane"],[null,null,3,"Sid","Sid"],[null,null,4,"null","null"]]`
	myToolId3NameAliceWant := `[["test@elastic.co","test@elastic.co",1,"Alice","Alice"],[null,null,3,"Sid","Sid"]]`
	myToolById4Want := `[[null,null,4,"null","null"]]`
	nullWant := `{"error":{"root_cause":[{"type":"verification_exception","reason":"Found 1 problem\nline 1:25: first argument of [name == ?name] is [text] so second argument must also be [text] but was [null]"}],"type":"verification_exception","reason":"Found 1 problem\nline 1:25: first argument of [name == ?name] is [text] so second argument must also be [text] but was [null]"},"status":400}`

	mcpSelect1Want := `{"jsonrpc":"2.0","id":"my-tool","result":{"content":[{"type":"text","text":"[[\"test@elastic.co\",\"test@elastic.co\",1,\"Alice\",\"Alice\"],[null,null,3,\"Sid\",\"Sid\"]]"}]}}`

	return select1Want, myToolId3NameAliceWant, myToolById4Want, nullWant, mcpSelect1Want
}

func getElasticsearchToolsConfig(sourceConfig map[string]any, toolKind, paramToolStatement, idParamToolStmt, nameParamToolStmt, arrayToolStatement, authToolStatement string) map[string]any {
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
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
				"query":       "FROM test-index | SORT id ASC",
			},
			"my-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with params.",
				"query":       paramToolStatement,
				"parameters": []any{
					map[string]any{
						"name":        "id",
						"type":        "integer",
						"description": "user ID",
					},
					map[string]any{
						"name":        "name",
						"type":        "string",
						"description": "user name",
					},
				},
			},
			"my-tool-by-id": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with params.",
				"query":       idParamToolStmt,
				"parameters": []any{
					map[string]any{
						"name":        "id",
						"type":        "integer",
						"description": "user ID",
					},
				},
			},
			"my-tool-by-name": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with params.",
				"query":       nameParamToolStmt,
				"parameters": []any{
					map[string]any{
						"name":        "name",
						"type":        "string",
						"description": "user name",
						"required":    false,
					},
				},
			},
			"my-array-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with array params.",
				"query":       arrayToolStatement,
				"parameters": []any{
					map[string]any{
						"name":        "idArray",
						"type":        "array",
						"description": "ID array",
						"items": map[string]any{
							"name":        "id",
							"type":        "integer",
							"description": "ID",
						},
					},
					map[string]any{
						"name":        "nameArray",
						"type":        "array",
						"description": "user name array",
						"items": map[string]any{
							"name":        "name",
							"type":        "string",
							"description": "user name",
						},
					},
				},
			},
			"my-auth-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test authenticated parameters.",
				// statement to auto-fill authenticated parameter
				"query": authToolStatement,
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
				"query":       "FROM *",
				"authRequired": []string{
					"my-google-auth",
				},
			},
			"my-fail-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test statement with incorrect syntax.",
				"query":       "SELEC 1;",
			},
		},
	}
	return toolsFile
}
