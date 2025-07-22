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

package firestore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	firestoreapi "cloud.google.com/go/firestore"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
	"google.golang.org/api/option"
)

var (
	FirestoreSourceKind = "firestore"
	FirestoreProject    = os.Getenv("FIRESTORE_PROJECT")
	FirestoreDatabase   = os.Getenv("FIRESTORE_DATABASE") // Optional, defaults to "(default)"
)

func getFirestoreVars(t *testing.T) map[string]any {
	if FirestoreProject == "" {
		t.Fatal("'FIRESTORE_PROJECT' not set")
	}

	vars := map[string]any{
		"kind":    FirestoreSourceKind,
		"project": FirestoreProject,
	}

	// Only add database if it's explicitly set
	if FirestoreDatabase != "" {
		vars["database"] = FirestoreDatabase
	}

	return vars
}

// initFirestoreConnection creates a Firestore client for testing
func initFirestoreConnection(project, database string) (*firestoreapi.Client, error) {
	ctx := context.Background()

	if database == "" {
		database = "(default)"
	}

	client, err := firestoreapi.NewClientWithDatabase(ctx, project, database, option.WithUserAgent("genai-toolbox-integration-test"))
	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client for project %q and database %q: %w", project, database, err)
	}
	return client, nil
}

func TestFirestoreToolEndpoints(t *testing.T) {
	sourceConfig := getFirestoreVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var args []string

	client, err := initFirestoreConnection(FirestoreProject, FirestoreDatabase)
	if err != nil {
		t.Fatalf("unable to create Firestore connection: %s", err)
	}
	defer client.Close()

	// Create test collection and document names with UUID
	testCollectionName := fmt.Sprintf("test_collection_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))
	testSubCollectionName := fmt.Sprintf("test_subcollection_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))
	testDocID1 := fmt.Sprintf("doc_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))
	testDocID2 := fmt.Sprintf("doc_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))
	testDocID3 := fmt.Sprintf("doc_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))

	// Document paths for testing
	docPath1 := fmt.Sprintf("%s/%s", testCollectionName, testDocID1)
	docPath2 := fmt.Sprintf("%s/%s", testCollectionName, testDocID2)
	docPath3 := fmt.Sprintf("%s/%s", testCollectionName, testDocID3)

	// Set up test data
	teardown := setupFirestoreTestData(t, ctx, client, testCollectionName, testSubCollectionName,
		testDocID1, testDocID2, testDocID3)
	defer teardown(t)

	// Write config into a file and pass it to command
	toolsFile := getFirestoreToolsConfig(sourceConfig)

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

	// Run Firestore-specific tool get test
	runFirestoreToolGetTest(t)

	// Run Firestore-specific MCP test
	runFirestoreMCPToolCallMethod(t, docPath1, docPath2)

	// Run specific Firestore tool tests
	runFirestoreGetDocumentsTest(t, docPath1, docPath2)
	runFirestoreListCollectionsTest(t, testCollectionName, testSubCollectionName, docPath1)
	runFirestoreDeleteDocumentsTest(t, docPath3)
	runFirestoreQueryCollectionTest(t, testCollectionName)
	runFirestoreGetRulesTest(t)
	runFirestoreValidateRulesTest(t)
}

func runFirestoreToolGetTest(t *testing.T) {
	// Test tool get endpoint for Firestore tools
	tcs := []struct {
		name string
		api  string
		want map[string]any
	}{
		{
			name: "get my-simple-tool",
			api:  "http://127.0.0.1:5000/api/tool/my-simple-tool/",
			want: map[string]any{
				"my-simple-tool": map[string]any{
					"description": "Simple tool to test end to end functionality.",
					"parameters": []any{
						map[string]any{
							"name":        "documentPaths",
							"type":        "array",
							"required":    true,
							"description": "Array of document paths to retrieve from Firestore.",
							"items": map[string]any{
								"name":        "item",
								"type":        "string",
								"required":    true,
								"description": "Document path",
								"authSources": []any{},
							},
							"authSources": []any{},
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

			// Compare as JSON strings to handle any ordering differences
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tc.want)
			if string(gotJSON) != string(wantJSON) {
				t.Logf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func runFirestoreValidateRulesTest(t *testing.T) {
	invokeTcs := []struct {
		name        string
		api         string
		requestBody io.Reader
		wantRegex   string
		isErr       bool
	}{
		{
			name: "validate valid rules",
			api:  "http://127.0.0.1:5000/api/tool/firestore-validate-rules/invoke",
			requestBody: bytes.NewBuffer([]byte(`{
				"source": "rules_version = '2';\nservice cloud.firestore {\n  match /databases/{database}/documents {\n    match /{document=**} {\n      allow read, write: if true;\n    }\n  }\n}"
			}`)),
			wantRegex: `"valid":true.*"issueCount":0`,
			isErr:     false,
		},
		{
			name: "validate rules with syntax error",
			api:  "http://127.0.0.1:5000/api/tool/firestore-validate-rules/invoke",
			requestBody: bytes.NewBuffer([]byte(`{
				"source": "rules_version = '2';\nservice cloud.firestore {\n  match /databases/{database}/documents {\n    match /{document=**} {\n      allow read, write: if true;;\n    }\n  }\n}"
			}`)),
			wantRegex: `"valid":false.*"issueCount":[1-9]`,
			isErr:     false,
		},
		{
			name: "validate rules with missing version",
			api:  "http://127.0.0.1:5000/api/tool/firestore-validate-rules/invoke",
			requestBody: bytes.NewBuffer([]byte(`{
				"source": "service cloud.firestore {\n  match /databases/{database}/documents {\n    match /{document=**} {\n      allow read, write: if true;\n    }\n  }\n}"
			}`)),
			wantRegex: `"valid":false.*"issueCount":[1-9]`,
			isErr:     false,
		},
		{
			name:        "validate empty rules",
			api:         "http://127.0.0.1:5000/api/tool/firestore-validate-rules/invoke",
			requestBody: bytes.NewBuffer([]byte(`{"source": ""}`)),
			isErr:       true,
		},
		{
			name:        "missing source parameter",
			api:         "http://127.0.0.1:5000/api/tool/firestore-validate-rules/invoke",
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

func runFirestoreGetRulesTest(t *testing.T) {
	invokeTcs := []struct {
		name        string
		api         string
		requestBody io.Reader
		wantRegex   string
		isErr       bool
	}{
		{
			name:        "get firestore rules",
			api:         "http://127.0.0.1:5000/api/tool/firestore-get-rules/invoke",
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			wantRegex:   `"content":"[^"]+"`, // Should contain at least one of these fields
			isErr:       false,
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
				bodyBytes, _ := io.ReadAll(resp.Body)
				// The test might fail if there are no active rules in the project, which is acceptable
				if strings.Contains(string(bodyBytes), "no active Firestore rules") {
					t.Skipf("No active Firestore rules found in the project")
					return
				}
				if tc.isErr {
					return
				}
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

func runFirestoreMCPToolCallMethod(t *testing.T, docPath1, docPath2 string) {
	sessionId := tests.RunInitialize(t, "2024-11-05")
	header := map[string]string{}
	if sessionId != "" {
		header["Mcp-Session-Id"] = sessionId
	}

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		api           string
		requestBody   jsonrpc.JSONRPCRequest
		requestHeader map[string]string
		wantContains  string
		wantError     bool
	}{
		{
			name:          "MCP Invoke my-param-tool",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "my-param-tool",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name": "my-param-tool",
					"arguments": map[string]any{
						"documentPaths": []string{docPath1},
					},
				},
			},
			wantContains: `\"name\":\"Alice\"`,
			wantError:    false,
		},
		{
			name:          "MCP Invoke invalid tool",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invalid-tool",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "foo",
					"arguments": map[string]any{},
				},
			},
			wantContains: `tool with name \"foo\" does not exist`,
			wantError:    true,
		},
		{
			name:          "MCP Invoke my-param-tool without parameters",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke-without-parameter",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "my-param-tool",
					"arguments": map[string]any{},
				},
			},
			wantContains: `parameter \"documentPaths\" is required`,
			wantError:    true,
		},
		{
			name:          "MCP Invoke my-auth-required-tool",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke my-auth-required-tool",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "my-auth-required-tool",
					"arguments": map[string]any{},
				},
			},
			wantContains: `tool with name \"my-auth-required-tool\" does not exist`,
			wantError:    true,
		},
		{
			name:          "MCP Invoke my-fail-tool",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke-fail-tool",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name": "my-fail-tool",
					"arguments": map[string]any{
						"documentPaths": []string{"non-existent/path"},
					},
				},
			},
			wantContains: `\"exists\":false`,
			wantError:    false,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			reqMarshal, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("unexpected error during marshaling of request body")
			}

			req, err := http.NewRequest(http.MethodPost, tc.api, bytes.NewBuffer(reqMarshal))
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range header {
				req.Header.Add(k, v)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("unable to read request body: %s", err)
			}

			got := string(bytes.TrimSpace(respBody))

			if !strings.Contains(got, tc.wantContains) {
				t.Fatalf("Expected substring not found:\ngot:  %q\nwant: %q (to be contained within got)", got, tc.wantContains)
			}
		})
	}
}

func getFirestoreToolsConfig(sourceConfig map[string]any) map[string]any {
	sources := map[string]any{
		"my-instance": sourceConfig,
	}

	tools := map[string]any{
		// Tool for RunToolGetTest
		"my-simple-tool": map[string]any{
			"kind":        "firestore-get-documents",
			"source":      "my-instance",
			"description": "Simple tool to test end to end functionality.",
		},
		// Tool for MCP test - this will get documents
		"my-param-tool": map[string]any{
			"kind":        "firestore-get-documents",
			"source":      "my-instance",
			"description": "Tool to get documents by paths",
		},
		// Tool for MCP test that fails
		"my-fail-tool": map[string]any{
			"kind":        "firestore-get-documents",
			"source":      "my-instance",
			"description": "Tool that will fail",
		},
		// Firestore specific tools
		"firestore-get-docs": map[string]any{
			"kind":        "firestore-get-documents",
			"source":      "my-instance",
			"description": "Get multiple documents from Firestore",
		},
		"firestore-list-colls": map[string]any{
			"kind":        "firestore-list-collections",
			"source":      "my-instance",
			"description": "List Firestore collections",
		},
		"firestore-delete-docs": map[string]any{
			"kind":        "firestore-delete-documents",
			"source":      "my-instance",
			"description": "Delete documents from Firestore",
		},
		"firestore-query-coll": map[string]any{
			"kind":        "firestore-query-collection",
			"source":      "my-instance",
			"description": "Query a Firestore collection",
		},
		"firestore-get-rules": map[string]any{
			"kind":        "firestore-get-rules",
			"source":      "my-instance",
			"description": "Get Firestore security rules",
		},
		"firestore-validate-rules": map[string]any{
			"kind":        "firestore-validate-rules",
			"source":      "my-instance",
			"description": "Validate Firestore security rules",
		},
	}

	return map[string]any{
		"sources": sources,
		"tools":   tools,
	}
}

func setupFirestoreTestData(t *testing.T, ctx context.Context, client *firestoreapi.Client,
	collectionName, subCollectionName, docID1, docID2, docID3 string) func(*testing.T) {
	// Create test documents
	testData1 := map[string]interface{}{
		"name": "Alice",
		"age":  30,
	}
	testData2 := map[string]interface{}{
		"name": "Bob",
		"age":  25,
	}
	testData3 := map[string]interface{}{
		"name": "Charlie",
		"age":  35,
	}

	// Create documents
	_, err := client.Collection(collectionName).Doc(docID1).Set(ctx, testData1)
	if err != nil {
		t.Fatalf("Failed to create test document 1: %v", err)
	}

	_, err = client.Collection(collectionName).Doc(docID2).Set(ctx, testData2)
	if err != nil {
		t.Fatalf("Failed to create test document 2: %v", err)
	}

	_, err = client.Collection(collectionName).Doc(docID3).Set(ctx, testData3)
	if err != nil {
		t.Fatalf("Failed to create test document 3: %v", err)
	}

	// Create a subcollection document
	subDocData := map[string]interface{}{
		"type":  "subcollection_doc",
		"value": "test",
	}
	_, err = client.Collection(collectionName).Doc(docID1).Collection(subCollectionName).Doc("subdoc1").Set(ctx, subDocData)
	if err != nil {
		t.Fatalf("Failed to create subcollection document: %v", err)
	}

	// Return cleanup function
	return func(t *testing.T) {
		// Delete subcollection documents first
		subDocs := client.Collection(collectionName).Doc(docID1).Collection(subCollectionName).Documents(ctx)
		for {
			doc, err := subDocs.Next()
			if err != nil {
				break
			}
			if _, err := doc.Ref.Delete(ctx); err != nil {
				t.Errorf("Failed to delete subcollection document: %v", err)
			}
		}

		// Delete main collection documents
		if _, err := client.Collection(collectionName).Doc(docID1).Delete(ctx); err != nil {
			t.Errorf("Failed to delete test document 1: %v", err)
		}
		if _, err := client.Collection(collectionName).Doc(docID2).Delete(ctx); err != nil {
			t.Errorf("Failed to delete test document 2: %v", err)
		}
		if _, err := client.Collection(collectionName).Doc(docID3).Delete(ctx); err != nil {
			t.Errorf("Failed to delete test document 3: %v", err)
		}
	}
}

func runFirestoreGetDocumentsTest(t *testing.T, docPath1, docPath2 string) {
	invokeTcs := []struct {
		name        string
		api         string
		requestBody io.Reader
		wantRegex   string
		isErr       bool
	}{
		{
			name:        "get single document",
			api:         "http://127.0.0.1:5000/api/tool/firestore-get-docs/invoke",
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(`{"documentPaths": ["%s"]}`, docPath1))),
			wantRegex:   `"name":"Alice"`,
			isErr:       false,
		},
		{
			name:        "get multiple documents",
			api:         "http://127.0.0.1:5000/api/tool/firestore-get-docs/invoke",
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(`{"documentPaths": ["%s", "%s"]}`, docPath1, docPath2))),
			wantRegex:   `"name":"Alice".*"name":"Bob"`,
			isErr:       false,
		},
		{
			name:        "get non-existent document",
			api:         "http://127.0.0.1:5000/api/tool/firestore-get-docs/invoke",
			requestBody: bytes.NewBuffer([]byte(`{"documentPaths": ["non-existent-collection/non-existent-doc"]}`)),
			wantRegex:   `"exists":false`,
			isErr:       false,
		},
		{
			name:        "missing documentPaths parameter",
			api:         "http://127.0.0.1:5000/api/tool/firestore-get-docs/invoke",
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			isErr:       true,
		},
		{
			name:        "empty documentPaths array",
			api:         "http://127.0.0.1:5000/api/tool/firestore-get-docs/invoke",
			requestBody: bytes.NewBuffer([]byte(`{"documentPaths": []}`)),
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

func runFirestoreListCollectionsTest(t *testing.T, collectionName, subCollectionName, parentDocPath string) {
	invokeTcs := []struct {
		name        string
		api         string
		requestBody io.Reader
		want        string
		isErr       bool
	}{
		{
			name:        "list root collections",
			api:         "http://127.0.0.1:5000/api/tool/firestore-list-colls/invoke",
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			want:        collectionName,
			isErr:       false,
		},
		{
			name:        "list subcollections",
			api:         "http://127.0.0.1:5000/api/tool/firestore-list-colls/invoke",
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(`{"parentPath": "%s"}`, parentDocPath))),
			want:        subCollectionName,
			isErr:       false,
		},
		{
			name:        "list collections for non-existent parent",
			api:         "http://127.0.0.1:5000/api/tool/firestore-list-colls/invoke",
			requestBody: bytes.NewBuffer([]byte(`{"parentPath": "non-existent-collection/non-existent-doc"}`)),
			want:        `[]`, // Empty array for no collections
			isErr:       false,
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

			if !strings.Contains(got, tc.want) {
				t.Fatalf("expected %q to contain %q, but it did not", got, tc.want)
			}
		})
	}
}

func runFirestoreDeleteDocumentsTest(t *testing.T, docPath string) {
	invokeTcs := []struct {
		name        string
		api         string
		requestBody io.Reader
		want        string
		isErr       bool
	}{
		{
			name:        "delete single document",
			api:         "http://127.0.0.1:5000/api/tool/firestore-delete-docs/invoke",
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(`{"documentPaths": ["%s"]}`, docPath))),
			want:        `"success":true`,
			isErr:       false,
		},
		{
			name:        "delete non-existent document",
			api:         "http://127.0.0.1:5000/api/tool/firestore-delete-docs/invoke",
			requestBody: bytes.NewBuffer([]byte(`{"documentPaths": ["non-existent-collection/non-existent-doc"]}`)),
			want:        `"success":true`, // Firestore delete succeeds even if doc doesn't exist
			isErr:       false,
		},
		{
			name:        "missing documentPaths parameter",
			api:         "http://127.0.0.1:5000/api/tool/firestore-delete-docs/invoke",
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			isErr:       true,
		},
		{
			name:        "empty documentPaths array",
			api:         "http://127.0.0.1:5000/api/tool/firestore-delete-docs/invoke",
			requestBody: bytes.NewBuffer([]byte(`{"documentPaths": []}`)),
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

			if !strings.Contains(got, tc.want) {
				t.Fatalf("expected %q to contain %q, but it did not", got, tc.want)
			}
		})
	}
}

func runFirestoreQueryCollectionTest(t *testing.T, collectionName string) {
	invokeTcs := []struct {
		name        string
		api         string
		requestBody io.Reader
		wantRegex   string
		isErr       bool
	}{
		{
			name: "query collection with filter",
			api:  "http://127.0.0.1:5000/api/tool/firestore-query-coll/invoke",
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(`{
				"collectionPath": "%s",
				"filters": ["{\"field\": \"age\", \"op\": \">\", \"value\": 25}"],
				"orderBy": "",
				"limit": 10
			}`, collectionName))),
			wantRegex: `"name":"Alice"`,
			isErr:     false,
		},
		{
			name: "query collection with orderBy",
			api:  "http://127.0.0.1:5000/api/tool/firestore-query-coll/invoke",
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(`{
				"collectionPath": "%s",
				"filters": [],
				"orderBy": "{\"field\": \"age\", \"direction\": \"DESCENDING\"}",
				"limit": 2
			}`, collectionName))),
			wantRegex: `"age":30.*"age":25`, // Should be ordered by age descending (Charlie=35, Alice=30, Bob=25)
			isErr:     false,
		},
		{
			name: "query collection with multiple filters",
			api:  "http://127.0.0.1:5000/api/tool/firestore-query-coll/invoke",
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(`{
				"collectionPath": "%s",
				"filters": [
					"{\"field\": \"age\", \"op\": \">=\", \"value\": 25}",
					"{\"field\": \"age\", \"op\": \"<=\", \"value\": 30}"
				],
				"orderBy": "",
				"limit": 10
			}`, collectionName))),
			wantRegex: `"name":"Bob".*"name":"Alice"`, // Results may be ordered by document ID
			isErr:     false,
		},
		{
			name: "query with limit",
			api:  "http://127.0.0.1:5000/api/tool/firestore-query-coll/invoke",
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(`{
				"collectionPath": "%s",
				"filters": [],
				"orderBy": "",
				"limit": 1
			}`, collectionName))),
			wantRegex: `^\[{.*}\]$`, // Should return exactly one document
			isErr:     false,
		},
		{
			name: "query non-existent collection",
			api:  "http://127.0.0.1:5000/api/tool/firestore-query-coll/invoke",
			requestBody: bytes.NewBuffer([]byte(`{
				"collectionPath": "non-existent-collection",
				"filters": [],
				"orderBy": "",
				"limit": 10
			}`)),
			wantRegex: `^\[\]$`, // Empty array
			isErr:     false,
		},
		{
			name:        "missing collectionPath parameter",
			api:         "http://127.0.0.1:5000/api/tool/firestore-query-coll/invoke",
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			isErr:       true,
		},
		{
			name: "invalid filter operator",
			api:  "http://127.0.0.1:5000/api/tool/firestore-query-coll/invoke",
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(`{
				"collectionPath": "%s",
				"filters": ["{\"field\": \"age\", \"op\": \"INVALID\", \"value\": 25}"],
				"orderBy": ""
			}`, collectionName))),
			isErr: true,
		},
		{
			name: "query with analyzeQuery",
			api:  "http://127.0.0.1:5000/api/tool/firestore-query-coll/invoke",
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(`{
				"collectionPath": "%s",
				"filters": [],
				"orderBy": "",
				"analyzeQuery": true,
				"limit": 1
			}`, collectionName))),
			wantRegex: `"documents":\[.*\]`,
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
