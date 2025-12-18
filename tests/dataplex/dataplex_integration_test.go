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
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	bigqueryapi "cloud.google.com/go/bigquery"
	dataplex "cloud.google.com/go/dataplex/apiv1"
	dataplexpb "cloud.google.com/go/dataplex/apiv1/dataplexpb"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var (
	DataplexSourceKind                = "dataplex"
	DataplexSearchEntriesToolKind     = "dataplex-search-entries"
	DataplexLookupEntryToolKind       = "dataplex-lookup-entry"
	DataplexSearchAspectTypesToolKind = "dataplex-search-aspect-types"
	DataplexProject                   = os.Getenv("DATAPLEX_PROJECT")
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

// Copied over from bigquery.go
func initBigQueryConnection(ctx context.Context, project string) (*bigqueryapi.Client, error) {
	cred, err := google.FindDefaultCredentials(ctx, bigqueryapi.Scope)
	if err != nil {
		return nil, fmt.Errorf("failed to find default Google Cloud credentials with scope %q: %w", bigqueryapi.Scope, err)
	}

	client, err := bigqueryapi.NewClient(ctx, project, option.WithCredentials(cred))
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client for project %q: %w", project, err)
	}
	return client, nil
}

func initDataplexConnection(ctx context.Context) (*dataplex.CatalogClient, error) {
	cred, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find default Google Cloud credentials: %w", err)
	}

	client, err := dataplex.NewCatalogClient(ctx, option.WithCredentials(cred))
	if err != nil {
		return nil, fmt.Errorf("failed to create Dataplex client %w", err)
	}
	return client, nil
}

func TestDataplexToolEndpoints(t *testing.T) {
	sourceConfig := getDataplexVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	var args []string

	bigqueryClient, err := initBigQueryConnection(ctx, DataplexProject)
	if err != nil {
		t.Fatalf("unable to create Cloud SQL connection pool: %s", err)
	}

	dataplexClient, err := initDataplexConnection(ctx)
	if err != nil {
		t.Fatalf("unable to create Dataplex connection: %s", err)
	}

	// create resources with UUID
	datasetName := fmt.Sprintf("temp_toolbox_test_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))
	tableName := fmt.Sprintf("param_table_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))
	aspectTypeId := fmt.Sprintf("param-aspect-type-%s", strings.ReplaceAll(uuid.New().String(), "-", ""))

	teardownTable1 := setupBigQueryTable(t, ctx, bigqueryClient, datasetName, tableName)
	teardownAspectType1 := setupDataplexThirdPartyAspectType(t, ctx, dataplexClient, aspectTypeId)
	time.Sleep(2 * time.Minute) // wait for table and aspect type to be ingested
	defer teardownTable1(t)
	defer teardownAspectType1(t)

	toolsFile := getDataplexToolsConfig(sourceConfig)

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	runDataplexToolGetTest(t)
	runDataplexSearchEntriesToolInvokeTest(t, tableName, datasetName)
	runDataplexLookupEntryToolInvokeTest(t, tableName, datasetName)
	runDataplexSearchAspectTypesToolInvokeTest(t, aspectTypeId)
}

func setupBigQueryTable(t *testing.T, ctx context.Context, client *bigqueryapi.Client, datasetName string, tableName string) func(*testing.T) {
	// Create dataset
	dataset := client.Dataset(datasetName)
	_, err := dataset.Metadata(ctx)

	if err != nil {
		apiErr, ok := err.(*googleapi.Error)
		if !ok || apiErr.Code != 404 {
			t.Fatalf("Failed to check dataset %q existence: %v", datasetName, err)
		}
		metadataToCreate := &bigqueryapi.DatasetMetadata{Name: datasetName}
		if err := dataset.Create(ctx, metadataToCreate); err != nil {
			t.Fatalf("Failed to create dataset %q: %v", datasetName, err)
		}
	}

	// Create table
	tab := client.Dataset(datasetName).Table(tableName)
	meta := &bigqueryapi.TableMetadata{}
	if err := tab.Create(ctx, meta); err != nil {
		t.Fatalf("Create table job for %s failed: %v", tableName, err)
	}

	return func(t *testing.T) {
		// tear down table
		dropSQL := fmt.Sprintf("drop table %s.%s", datasetName, tableName)
		dropJob, err := client.Query(dropSQL).Run(ctx)
		if err != nil {
			t.Errorf("Failed to start drop table job for %s: %v", tableName, err)
			return
		}
		dropStatus, err := dropJob.Wait(ctx)
		if err != nil {
			t.Errorf("Failed to wait for drop table job for %s: %v", tableName, err)
			return
		}
		if err := dropStatus.Err(); err != nil {
			t.Errorf("Error dropping table %s: %v", tableName, err)
		}

		// tear down dataset
		datasetToTeardown := client.Dataset(datasetName)
		tablesIterator := datasetToTeardown.Tables(ctx)
		_, err = tablesIterator.Next()

		if err == iterator.Done {
			if err := datasetToTeardown.Delete(ctx); err != nil {
				t.Errorf("Failed to delete dataset %s: %v", datasetName, err)
			}
		} else if err != nil {
			t.Errorf("Failed to list tables in dataset %s to check emptiness: %v.", datasetName, err)
		}
	}
}

func setupDataplexThirdPartyAspectType(t *testing.T, ctx context.Context, client *dataplex.CatalogClient, aspectTypeId string) func(*testing.T) {
	parent := fmt.Sprintf("projects/%s/locations/us", DataplexProject)
	createAspectTypeReq := &dataplexpb.CreateAspectTypeRequest{
		Parent:       parent,
		AspectTypeId: aspectTypeId,
		AspectType: &dataplexpb.AspectType{
			Name: fmt.Sprintf("%s/aspectTypes/%s", parent, aspectTypeId),
			MetadataTemplate: &dataplexpb.AspectType_MetadataTemplate{
				Name: "UserSchema",
				Type: "record",
			},
		},
	}
	_, err := client.CreateAspectType(ctx, createAspectTypeReq)
	if err != nil {
		t.Fatalf("Failed to create aspect type %s: %v", aspectTypeId, err)
	}

	return func(t *testing.T) {
		// tear down aspect type
		deleteAspectTypeReq := &dataplexpb.DeleteAspectTypeRequest{
			Name: fmt.Sprintf("%s/aspectTypes/%s", parent, aspectTypeId),
		}
		if _, err := client.DeleteAspectType(ctx, deleteAspectTypeReq); err != nil {
			t.Errorf("Failed to delete aspect type %s: %v", aspectTypeId, err)
		}
	}
}

func getDataplexToolsConfig(sourceConfig map[string]any) map[string]any {
	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-dataplex-instance": sourceConfig,
		},
		"authServices": map[string]any{
			"my-google-auth": map[string]any{
				"kind":     "google",
				"clientId": tests.ClientId,
			},
		},
		"tools": map[string]any{
			"my-dataplex-search-entries-tool": map[string]any{
				"kind":        DataplexSearchEntriesToolKind,
				"source":      "my-dataplex-instance",
				"description": "Simple dataplex search entries tool to test end to end functionality.",
			},
			"my-auth-dataplex-search-entries-tool": map[string]any{
				"kind":         DataplexSearchEntriesToolKind,
				"source":       "my-dataplex-instance",
				"description":  "Simple dataplex search entries tool to test end to end functionality.",
				"authRequired": []string{"my-google-auth"},
			},
			"my-dataplex-lookup-entry-tool": map[string]any{
				"kind":        DataplexLookupEntryToolKind,
				"source":      "my-dataplex-instance",
				"description": "Simple dataplex lookup entry tool to test end to end functionality.",
			},
			"my-auth-dataplex-lookup-entry-tool": map[string]any{
				"kind":         DataplexLookupEntryToolKind,
				"source":       "my-dataplex-instance",
				"description":  "Simple dataplex lookup entry tool to test end to end functionality.",
				"authRequired": []string{"my-google-auth"},
			},
			"my-dataplex-search-aspect-types-tool": map[string]any{
				"kind":        DataplexSearchAspectTypesToolKind,
				"source":      "my-dataplex-instance",
				"description": "Simple dataplex search aspect types tool to test end to end functionality.",
			},
			"my-auth-dataplex-search-aspect-types-tool": map[string]any{
				"kind":         DataplexSearchAspectTypesToolKind,
				"source":       "my-dataplex-instance",
				"description":  "Simple dataplex search aspect types tool to test end to end functionality.",
				"authRequired": []string{"my-google-auth"},
			},
		},
	}

	return toolsFile
}

func runDataplexToolGetTest(t *testing.T) {
	testCases := []struct {
		name           string
		toolName       string
		expectedParams []string
	}{
		{
			name:           "get my-dataplex-search-entries-tool",
			toolName:       "my-dataplex-search-entries-tool",
			expectedParams: []string{"pageSize", "query", "orderBy"},
		},
		{
			name:           "get my-dataplex-lookup-entry-tool",
			toolName:       "my-dataplex-lookup-entry-tool",
			expectedParams: []string{"name", "view", "aspectTypes", "entry"},
		},
		{
			name:           "get my-dataplex-search-aspect-types-tool",
			toolName:       "my-dataplex-search-aspect-types-tool",
			expectedParams: []string{"pageSize", "query", "orderBy"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:5000/api/tool/%s/", tc.toolName))
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

			toolsMap, ok := got.(map[string]interface{})
			if !ok {
				t.Fatalf("expected 'tools' to be a map, got %T", got)
			}
			tool, ok := toolsMap[tc.toolName].(map[string]interface{})
			if !ok {
				t.Fatalf("expected tool %q to be a map, got %T", tc.toolName, toolsMap[tc.toolName])
			}
			params, ok := tool["parameters"].([]interface{})
			if !ok {
				t.Fatalf("expected 'parameters' to be a slice, got %T", tool["parameters"])
			}
			paramSet := make(map[string]struct{})
			for _, param := range params {
				paramMap, ok := param.(map[string]interface{})
				if ok {
					if name, ok := paramMap["name"].(string); ok {
						paramSet[name] = struct{}{}
					}
				}
			}
			var missing []string
			for _, want := range tc.expectedParams {
				if _, found := paramSet[want]; !found {
					missing = append(missing, want)
				}
			}
			if len(missing) > 0 {
				t.Fatalf("missing parameters for tool %q: %v", tc.toolName, missing)
			}
		})
	}
}

func runDataplexSearchEntriesToolInvokeTest(t *testing.T, tableName string, datasetName string) {
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	testCases := []struct {
		name           string
		api            string
		requestHeader  map[string]string
		requestBody    io.Reader
		wantStatusCode int
		expectResult   bool
		wantContentKey string
	}{
		{
			name:           "Success - Entry Found",
			api:            "http://127.0.0.1:5000/api/tool/my-dataplex-search-entries-tool/invoke",
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf("{\"query\":\"displayname=%s system=bigquery parent:%s\"}", tableName, datasetName))),
			wantStatusCode: 200,
			expectResult:   true,
			wantContentKey: "dataplex_entry",
		},
		{
			name:           "Success with Authorization - Entry Found",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-dataplex-search-entries-tool/invoke",
			requestHeader:  map[string]string{"my-google-auth_token": idToken},
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf("{\"query\":\"displayname=%s system=bigquery parent:%s\"}", tableName, datasetName))),
			wantStatusCode: 200,
			expectResult:   true,
			wantContentKey: "dataplex_entry",
		},
		{
			name:           "Failure - Invalid Authorization Token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-dataplex-search-entries-tool/invoke",
			requestHeader:  map[string]string{"my-google-auth_token": "invalid_token"},
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf("{\"query\":\"displayname=%s system=bigquery parent:%s\"}", tableName, datasetName))),
			wantStatusCode: 401,
			expectResult:   false,
			wantContentKey: "dataplex_entry",
		},
		{
			name:           "Failure - Without Authorization Token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-dataplex-search-entries-tool/invoke",
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf("{\"query\":\"displayname=%s system=bigquery parent:%s\"}", tableName, datasetName))),
			wantStatusCode: 401,
			expectResult:   false,
			wantContentKey: "dataplex_entry",
		},
		{
			name:           "Failure - Entry Not Found",
			api:            "http://127.0.0.1:5000/api/tool/my-dataplex-search-entries-tool/invoke",
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(`{"query":"displayname=\"\" system=bigquery parent:\"\""}`)),
			wantStatusCode: 200,
			expectResult:   false,
			wantContentKey: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("response status code is not %d. It is %d", tc.wantStatusCode, resp.StatusCode)
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("Response body: %s", string(bodyBytes))
			}
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("error parsing response body: %s", err)
			}
			resultStr, ok := result["result"].(string)
			if !ok {
				if result["result"] == nil && !tc.expectResult {
					return
				}
				t.Fatalf("expected 'result' field to be a string, got %T", result["result"])
			}
			if !tc.expectResult && (resultStr == "" || resultStr == "[]") {
				return
			}
			var entries []interface{}
			if err := json.Unmarshal([]byte(resultStr), &entries); err != nil {
				t.Fatalf("error unmarshalling result string: %v", err)
			}

			if tc.expectResult {
				if len(entries) != 1 {
					t.Fatalf("expected exactly one entry, but got %d", len(entries))
				}
				entry, ok := entries[0].(map[string]interface{})
				if !ok {
					t.Fatalf("expected first entry to be a map, got %T", entries[0])
				}
				if _, ok := entry[tc.wantContentKey]; !ok {
					t.Fatalf("expected entry to have key '%s', but it was not found in %v", tc.wantContentKey, entry)
				}
			} else {
				if len(entries) != 0 {
					t.Fatalf("expected 0 entries, but got %d", len(entries))
				}
			}
		})
	}
}

func runDataplexLookupEntryToolInvokeTest(t *testing.T, tableName string, datasetName string) {
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	testCases := []struct {
		name               string
		wantStatusCode     int
		api                string
		requestHeader      map[string]string
		requestBody        io.Reader
		expectResult       bool
		wantContentKey     string
		dontWantContentKey string
		aspectCheck        bool
		reqBodyMap         map[string]any
	}{
		{
			name:           "Success - Entry Found",
			api:            "http://127.0.0.1:5000/api/tool/my-dataplex-lookup-entry-tool/invoke",
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf("{\"name\":\"projects/%s/locations/us\", \"entry\":\"projects/%s/locations/us/entryGroups/@bigquery/entries/bigquery.googleapis.com/projects/%s/datasets/%s\"}", DataplexProject, DataplexProject, DataplexProject, datasetName))),
			wantStatusCode: 200,
			expectResult:   true,
			wantContentKey: "name",
		},
		{
			name:           "Success - Entry Found with Authorization",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-dataplex-lookup-entry-tool/invoke",
			requestHeader:  map[string]string{"my-google-auth_token": idToken},
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf("{\"name\":\"projects/%s/locations/us\", \"entry\":\"projects/%s/locations/us/entryGroups/@bigquery/entries/bigquery.googleapis.com/projects/%s/datasets/%s\"}", DataplexProject, DataplexProject, DataplexProject, datasetName))),
			wantStatusCode: 200,
			expectResult:   true,
			wantContentKey: "name",
		},
		{
			name:           "Failure - Invalid Authorization Token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-dataplex-lookup-entry-tool/invoke",
			requestHeader:  map[string]string{"my-google-auth_token": "invalid_token"},
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf("{\"name\":\"projects/%s/locations/us\", \"entry\":\"projects/%s/locations/us/entryGroups/@bigquery/entries/bigquery.googleapis.com/projects/%s/datasets/%s\"}", DataplexProject, DataplexProject, DataplexProject, datasetName))),
			wantStatusCode: 401,
			expectResult:   false,
			wantContentKey: "name",
		},
		{
			name:           "Failure - Without Authorization Token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-dataplex-lookup-entry-tool/invoke",
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf("{\"name\":\"projects/%s/locations/us\", \"entry\":\"projects/%s/locations/us/entryGroups/@bigquery/entries/bigquery.googleapis.com/projects/%s/datasets/%s\"}", DataplexProject, DataplexProject, DataplexProject, datasetName))),
			wantStatusCode: 401,
			expectResult:   false,
			wantContentKey: "name",
		},
		{
			name:           "Failure - Entry Not Found or Permission Denied",
			api:            "http://127.0.0.1:5000/api/tool/my-dataplex-lookup-entry-tool/invoke",
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf("{\"name\":\"projects/%s/locations/us\", \"entry\":\"projects/%s/locations/us/entryGroups/@bigquery/entries/bigquery.googleapis.com/projects/%s/datasets/%s\"}", DataplexProject, DataplexProject, DataplexProject, "non-existent-dataset"))),
			wantStatusCode: 400,
			expectResult:   false,
		},
		{
			name:               "Success - Entry Found with Basic View",
			api:                "http://127.0.0.1:5000/api/tool/my-dataplex-lookup-entry-tool/invoke",
			requestHeader:      map[string]string{},
			requestBody:        bytes.NewBuffer([]byte(fmt.Sprintf("{\"name\":\"projects/%s/locations/us\", \"entry\":\"projects/%s/locations/us/entryGroups/@bigquery/entries/bigquery.googleapis.com/projects/%s/datasets/%s/tables/%s\", \"view\": %d}", DataplexProject, DataplexProject, DataplexProject, datasetName, tableName, 1))),
			wantStatusCode:     200,
			expectResult:       true,
			wantContentKey:     "name",
			dontWantContentKey: "aspects",
		},
		{
			name:           "Failure - Entry with Custom View without Aspect Types",
			api:            "http://127.0.0.1:5000/api/tool/my-dataplex-lookup-entry-tool/invoke",
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf("{\"name\":\"projects/%s/locations/us\", \"entry\":\"projects/%s/locations/us/entryGroups/@bigquery/entries/bigquery.googleapis.com/projects/%s/datasets/%s/tables/%s\", \"view\": %d}", DataplexProject, DataplexProject, DataplexProject, datasetName, tableName, 3))),
			wantStatusCode: 400,
			expectResult:   false,
		},
		{
			name:           "Success - Entry Found with only Schema Aspect",
			api:            "http://127.0.0.1:5000/api/tool/my-dataplex-lookup-entry-tool/invoke",
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf("{\"name\":\"projects/%s/locations/us\", \"entry\":\"projects/%s/locations/us/entryGroups/@bigquery/entries/bigquery.googleapis.com/projects/%s/datasets/%s/tables/%s\", \"aspectTypes\":[\"projects/dataplex-types/locations/global/aspectTypes/schema\"], \"view\": %d}", DataplexProject, DataplexProject, DataplexProject, datasetName, tableName, 3))),
			wantStatusCode: 200,
			expectResult:   true,
			wantContentKey: "aspects",
			aspectCheck:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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

			if resp.StatusCode != tc.wantStatusCode {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("Response status code got %d, want %d\nResponse body: %s", resp.StatusCode, tc.wantStatusCode, string(bodyBytes))
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("Error parsing response body: %v", err)
			}

			if tc.expectResult {
				resultStr, ok := result["result"].(string)
				if !ok {
					t.Fatalf("Expected 'result' field to be a string on success, got %T", result["result"])
				}
				if resultStr == "" || resultStr == "{}" || resultStr == "null" {
					t.Fatal("Expected an entry, but got empty result")
				}

				var entry map[string]interface{}
				if err := json.Unmarshal([]byte(resultStr), &entry); err != nil {
					t.Fatalf("Error unmarshalling result string into entry map: %v", err)
				}

				if _, ok := entry[tc.wantContentKey]; !ok {
					t.Fatalf("Expected entry to have key '%s', but it was not found in %v", tc.wantContentKey, entry)
				}

				if _, ok := entry[tc.dontWantContentKey]; ok {
					t.Fatalf("Expected entry to not have key '%s', but it was found in %v", tc.dontWantContentKey, entry)
				}

				if tc.aspectCheck {
					// Check length of aspects
					aspects, ok := entry["aspects"].(map[string]interface{})
					if !ok {
						t.Fatalf("Expected 'aspects' to be a map, got %T", aspects)
					}
					if len(aspects) != 1 {
						t.Fatalf("Expected exactly one aspect, but got %d", len(aspects))
					}
				}
			} else { // Handle expected error response
				_, ok := result["error"]
				if !ok {
					t.Fatalf("Expected 'error' field in response, got %v", result)
				}
			}
		})
	}
}

func runDataplexSearchAspectTypesToolInvokeTest(t *testing.T, aspectTypeId string) {
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	testCases := []struct {
		name           string
		api            string
		requestHeader  map[string]string
		requestBody    io.Reader
		wantStatusCode int
		expectResult   bool
		wantContentKey string
	}{
		{
			name:           "Success - Aspect Type Found",
			api:            "http://127.0.0.1:5000/api/tool/my-dataplex-search-aspect-types-tool/invoke",
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf("{\"query\":\"name:%s_aspectType\"}", aspectTypeId))),
			wantStatusCode: 200,
			expectResult:   true,
			wantContentKey: "metadata_template",
		},
		{
			name:           "Success - Aspect Type Found with Authorization",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-dataplex-search-aspect-types-tool/invoke",
			requestHeader:  map[string]string{"my-google-auth_token": idToken},
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf("{\"query\":\"name:%s_aspectType\"}", aspectTypeId))),
			wantStatusCode: 200,
			expectResult:   true,
			wantContentKey: "metadata_template",
		},
		{
			name:           "Failure - Aspect Type Not Found",
			api:            "http://127.0.0.1:5000/api/tool/my-dataplex-search-aspect-types-tool/invoke",
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(`"{\"query\":\"name:_aspectType\"}"`)),
			wantStatusCode: 400,
			expectResult:   false,
		},
		{
			name:           "Failure - Invalid Authorization Token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-dataplex-search-aspect-types-tool/invoke",
			requestHeader:  map[string]string{"my-google-auth_token": "invalid_token"},
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf("{\"query\":\"name:%s_aspectType\"}", aspectTypeId))),
			wantStatusCode: 401,
			expectResult:   false,
		},
		{
			name:           "Failure - No Authorization Token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-dataplex-search-aspect-types-tool/invoke",
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf("{\"query\":\"name:%s_aspectType\"}", aspectTypeId))),
			wantStatusCode: 401,
			expectResult:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("response status code is not %d. It is %d", tc.wantStatusCode, resp.StatusCode)
			}
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("error parsing response body: %s", err)
			}
			resultStr, ok := result["result"].(string)
			if !ok {
				if result["result"] == nil && !tc.expectResult {
					return
				}
				t.Fatalf("expected 'result' field to be a string, got %T", result["result"])
			}
			if !tc.expectResult && (resultStr == "" || resultStr == "[]") {
				return
			}
			var entries []interface{}
			if err := json.Unmarshal([]byte(resultStr), &entries); err != nil {
				t.Fatalf("error unmarshalling result string: %v", err)
			}

			if tc.expectResult {
				if len(entries) != 1 {
					t.Fatalf("expected exactly one entry, but got %d", len(entries))
				}
				entry, ok := entries[0].(map[string]interface{})
				if !ok {
					t.Fatalf("expected entry to be a map, got %T", entries[0])
				}
				if _, ok := entry[tc.wantContentKey]; !ok {
					t.Fatalf("expected entry to have key '%s', but it was not found in %v", tc.wantContentKey, entry)
				}
			} else {
				if len(entries) != 0 {
					t.Fatalf("expected 0 entries, but got %d", len(entries))
				}
			}
		})
	}
}
