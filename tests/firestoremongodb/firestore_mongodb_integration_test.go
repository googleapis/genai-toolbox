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

package firestoremongodb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	firestoreapi "cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"
	"google.golang.org/api/option"
)

var (
	FirestoreSourceType      = "firestore"
	FirestoreMongodbProject  = os.Getenv("FIRESTORE_MONGODB_PROJECT")
	FirestoreMongodbDatabase = os.Getenv("FIRESTORE_MONGODB_DATABASE")
)

func getFirestoreMongodbVars(t *testing.T) map[string]any {
	project := FirestoreMongodbProject
	if project == "" {
		project = os.Getenv("FIRESTORE_PROJECT")
	}
	if project == "" {
		t.Fatal("'FIRESTORE_MONGODB_PROJECT' or 'FIRESTORE_PROJECT' not set")
	}

	database := FirestoreMongodbDatabase
	if database == "" {
		database = "mcp-toolbox-db-native-schema"
	}

	vars := map[string]any{
		"type":     FirestoreSourceType,
		"project":  project,
		"database": database,
	}

	return vars
}

// initFirestoreConnection creates a Firestore client for testing
func initFirestoreConnection(project, database string) (*firestoreapi.Client, error) {
	ctx := context.Background()

	if database == "" {
		database = "mcp-toolbox-db-native-schema"
	}

	client, err := firestoreapi.NewClientWithDatabase(ctx, project, database, option.WithUserAgent("genai-toolbox-integration-test"))
	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client for project %q and database %q: %w", project, database, err)
	}
	return client, nil
}

func TestFirestoreMongodbToolEndpoints(t *testing.T) {
	sourceConfig := getFirestoreMongodbVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	args := []string{"--enable-api"}

	project := sourceConfig["project"].(string)
	database := sourceConfig["database"].(string)

	client, err := initFirestoreConnection(project, database)
	if err != nil {
		t.Fatalf("unable to create Firestore connection: %s", err)
	}
	defer client.Close()

	// Create test collection and document names with UUID
	testCollectionName := fmt.Sprintf("test_collection_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))
	testDocID1 := fmt.Sprintf("doc_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))
	testDocID2 := fmt.Sprintf("doc_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))

	// Set up test data
	teardown := setupFirestoreMongodbTestData(t, ctx, client, testCollectionName, testDocID1, testDocID2)
	defer teardown(t)

	// Write config into a file and pass it to command
	toolsFile := getFirestoreMongodbToolsConfig(sourceConfig)

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

	// Run tool get tests
	runFirestoreMongodbToolGetTest(t)

	// Run tool execution tests
	runFirestoreMongodbGetSchemaTest(t, testCollectionName)
	runFirestoreMongodbExecuteMQLTest(t, testCollectionName)
}

func runFirestoreMongodbToolGetTest(t *testing.T) {
	tcs := []struct {
		name string
		api  string
		want map[string]any
	}{
		{
			name: "get firestore-mongodb-get-schema",
			api:  "http://127.0.0.1:5000/api/tool/firestore-mongodb-get-schema/",
			want: map[string]any{
				"firestore-mongodb-get-schema": map[string]any{
					"description": "Get schema for Firestore collections",
					"parameters": []any{
						map[string]any{
							"name":         "collection",
							"type":         "string",
							"required":     false,
							"default":      "",
							"description":  "Optional name or path of a specific collection to get schema for. If omitted, schemas for all root collections are returned.",
							"authServices": []any{},
						},
					},
					"authRequired": []any{},
				},
			},
		},
		{
			name: "get firestore-mongodb-execute-mql",
			api:  "http://127.0.0.1:5000/api/tool/firestore-mongodb-execute-mql/",
			want: map[string]any{
				"firestore-mongodb-execute-mql": map[string]any{
					"description": "Execute MQL query or aggregation pipeline against Firestore",
					"parameters": []any{
						map[string]any{
							"name":         "query",
							"type":         "string",
							"required":     true,
							"description":  "The MQL query or aggregation pipeline to execute against Firestore.",
							"authServices": []any{},
						},
					},
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

			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tc.want)
			if string(gotJSON) != string(wantJSON) {
				t.Logf("got %v, want %v", got, tc.want)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func getFirestoreMongodbToolsConfig(sourceConfig map[string]any) map[string]any {
	sources := map[string]any{
		"my-instance": sourceConfig,
	}

	tools := map[string]any{
		"firestore-mongodb-get-schema": map[string]any{
			"type":        "firestore-mongodb-get-schema",
			"source":      "my-instance",
			"description": "Get schema for Firestore collections",
		},
		"firestore-mongodb-execute-mql": map[string]any{
			"type":        "firestore-mongodb-execute-mql",
			"source":      "my-instance",
			"description": "Execute MQL query or aggregation pipeline against Firestore",
		},
	}

	return map[string]any{
		"sources": sources,
		"tools":   tools,
	}
}

func setupFirestoreMongodbTestData(t *testing.T, ctx context.Context, client *firestoreapi.Client,
	collectionName, docID1, docID2 string) func(*testing.T) {
	testData1 := map[string]interface{}{
		"name":  "Alice",
		"age":   30,
		"score": 95.5,
	}
	testData2 := map[string]interface{}{
		"name":   "Bob",
		"age":    25,
		"active": true,
	}

	_, err := client.Collection(collectionName).Doc(docID1).Set(ctx, testData1)
	if err != nil {
		t.Fatalf("Failed to create test document 1: %v", err)
	}

	_, err = client.Collection(collectionName).Doc(docID2).Set(ctx, testData2)
	if err != nil {
		t.Fatalf("Failed to create test document 2: %v", err)
	}

	return func(t *testing.T) {
		// Clean up only the documents created for this test collection
		_, err := client.Collection(collectionName).Doc(docID1).Delete(ctx)
		if err != nil {
			t.Logf("Failed to delete test document 1: %v", err)
		}
		_, err = client.Collection(collectionName).Doc(docID2).Delete(ctx)
		if err != nil {
			t.Logf("Failed to delete test document 2: %v", err)
		}
	}
}

func runFirestoreMongodbGetSchemaTest(t *testing.T, collectionName string) {
	invokeTcs := []struct {
		name        string
		api         string
		requestBody io.Reader
		wantRegex   string
		isErr       bool
	}{
		{
			name: "get schema for specific collection",
			api:  "http://127.0.0.1:5000/api/tool/firestore-mongodb-get-schema/invoke",
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(`{
				"collection": "%s"
			}`, collectionName))),
			wantRegex: fmt.Sprintf(`"collection":"%s"`, collectionName),
			isErr:     false,
		},
		{
			name:        "get schema for all root collections",
			api:         "http://127.0.0.1:5000/api/tool/firestore-mongodb-get-schema/invoke",
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			wantRegex:   fmt.Sprintf(`"collection":"%s"`, collectionName),
			isErr:       false,
		},
		{
			name: "get schema for non-existent collection",
			api:  "http://127.0.0.1:5000/api/tool/firestore-mongodb-get-schema/invoke",
			requestBody: bytes.NewBuffer([]byte(`{
				"collection": "non_existent_collection_xyz"
			}`)),
			wantRegex: `.*`,
			isErr:     false,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")

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

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body: %v", err)
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if tc.wantRegex != "" {
				matched, err := regexp.MatchString(tc.wantRegex, got)
				if err != nil {
					t.Fatalf("invalid regex pattern: %v", err)
				}
				if !matched {
					t.Fatalf("result does not match expected pattern.\nGot: %s\nWant pattern: %s", got, tc.wantRegex)
				}
			}
		})
	}
}

func runFirestoreMongodbExecuteMQLTest(t *testing.T, collectionName string) {
	invokeTcs := []struct {
		name        string
		api         string
		requestBody io.Reader
		wantRegex   string
		isErr       bool
	}{
		{
			name: "execute MQL structured pipeline get_schema stage",
			api:  "http://127.0.0.1:5000/api/tool/firestore-mongodb-execute-mql/invoke",
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(`{
				"query": "{\"structuredPipeline\": {\"pipeline\": {\"stages\": [{\"name\": \"get_schema\", \"args\": [{\"stringValue\": \"{\\\"collection\\\": \\\"%s\\\", \\\"semantics\\\": \\\"mongodb\\\"}\"}]}]}}}"
			}`, collectionName))),
			wantRegex: `.*`,
			isErr:     false,
		},
		{
			name: "execute MQL find query",
			api:  "http://127.0.0.1:5000/api/tool/firestore-mongodb-execute-mql/invoke",
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(`{
				"query": "%s.find({\"name\": \"Alice\"})"
			}`, collectionName))),
			wantRegex: `.*`,
			isErr:     false,
		},
		{
			name:        "execute MQL with empty query",
			api:         "http://127.0.0.1:5000/api/tool/firestore-mongodb-execute-mql/invoke",
			requestBody: bytes.NewBuffer([]byte(`{"query": ""}`)),
			isErr:       true,
		},
		{
			name:        "missing query parameter",
			api:         "http://127.0.0.1:5000/api/tool/firestore-mongodb-execute-mql/invoke",
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			isErr:       true,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")

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

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body: %v", err)
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if tc.wantRegex != "" {
				matched, err := regexp.MatchString(tc.wantRegex, got)
				if err != nil {
					t.Fatalf("invalid regex pattern: %v", err)
				}
				if !matched {
					t.Fatalf("result does not match expected pattern.\nGot: %s\nWant pattern: %s", got, tc.wantRegex)
				}
			}
		})
	}
}
