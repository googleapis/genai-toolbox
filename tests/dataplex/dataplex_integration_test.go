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
	DataplexSourceType                = "dataplex"
	DataplexSearchEntriesToolType     = "dataplex-search-entries"
	DataplexLookupEntryToolType       = "dataplex-lookup-entry"
	DataplexSearchAspectTypesToolType = "dataplex-search-aspect-types"
	DataplexProject                   = os.Getenv("DATAPLEX_PROJECT")
)

func getDataplexVars(t *testing.T) map[string]any {
	switch "" {
	case DataplexProject:
		t.Skip("'DATAPLEX_PROJECT' not set")
	}
	return map[string]any{
		"type":    DataplexSourceType,
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

// cleanupOldAspectTypes Deletes AspectTypes older than the specified duration.
func cleanupOldAspectTypes(t *testing.T, ctx context.Context, client *dataplex.CatalogClient, oldThreshold time.Duration) {
	parent := fmt.Sprintf("projects/%s/locations/us", DataplexProject)
	olderThanTime := time.Now().Add(-oldThreshold)

	listReq := &dataplexpb.ListAspectTypesRequest{
		Parent:   parent,
		PageSize: 100,               // Fetch up to 100 items
		OrderBy:  "create_time asc", // Order by creation time
	}

	const maxDeletes = 8 // Explicitly limit the number of deletions
	it := client.ListAspectTypes(ctx, listReq)
	var aspectTypesToDelete []string
	for len(aspectTypesToDelete) < maxDeletes {
		aspectType, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Logf("Warning: Failed to list aspect types during cleanup: %v", err)
			return
		}
		// Perform time-based filtering in memory
		if aspectType.CreateTime != nil {
			createTime := aspectType.CreateTime.AsTime()
			if createTime.Before(olderThanTime) {
				aspectTypesToDelete = append(aspectTypesToDelete, aspectType.GetName())
			}
		} else {
			t.Logf("Warning: AspectType %s has no CreateTime", aspectType.GetName())
		}
	}
	if len(aspectTypesToDelete) == 0 {
		t.Logf("cleanupOldAspectTypes: No aspect types found older than %s to delete.", oldThreshold.String())
		return
	}

	for _, aspectTypeName := range aspectTypesToDelete {
		deleteReq := &dataplexpb.DeleteAspectTypeRequest{Name: aspectTypeName}
		op, err := client.DeleteAspectType(ctx, deleteReq)
		if err != nil {
			t.Logf("Warning: Failed to delete aspect type %s: %v", aspectTypeName, err)
			continue // Skip to the next item if initiation fails
		}

		if err := op.Wait(ctx); err != nil {
			t.Logf("Warning: Failed to delete aspect type %s, operation error: %v", aspectTypeName, err)
		} else {
			t.Logf("cleanupOldAspectTypes: Successfully deleted %s", aspectTypeName)
		}
	}
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

	// Cleanup older aspecttypes
	cleanupOldAspectTypes(t, ctx, dataplexClient, 1*time.Hour)

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
				"type":     "google",
				"clientId": tests.ClientId,
			},
		},
		"tools": map[string]any{
			"my-dataplex-search-entries-tool": map[string]any{
				"type":        DataplexSearchEntriesToolType,
				"source":      "my-dataplex-instance",
				"description": "Simple dataplex search entries tool to test end to end functionality.",
			},
			"my-auth-dataplex-search-entries-tool": map[string]any{
				"type":         DataplexSearchEntriesToolType,
				"source":       "my-dataplex-instance",
				"description":  "Simple dataplex search entries tool to test end to end functionality.",
				"authRequired": []string{"my-google-auth"},
			},
			"my-dataplex-lookup-entry-tool": map[string]any{
				"type":        DataplexLookupEntryToolType,
				"source":      "my-dataplex-instance",
				"description": "Simple dataplex lookup entry tool to test end to end functionality.",
			},
			"my-auth-dataplex-lookup-entry-tool": map[string]any{
				"type":         DataplexLookupEntryToolType,
				"source":       "my-dataplex-instance",
				"description":  "Simple dataplex lookup entry tool to test end to end functionality.",
				"authRequired": []string{"my-google-auth"},
			},
			"my-dataplex-search-aspect-types-tool": map[string]any{
				"type":        DataplexSearchAspectTypesToolType,
				"source":      "my-dataplex-instance",
				"description": "Simple dataplex search aspect types tool to test end to end functionality.",
			},
			"my-auth-dataplex-search-aspect-types-tool": map[string]any{
				"type":         DataplexSearchAspectTypesToolType,
				"source":       "my-dataplex-instance",
				"description":  "Simple dataplex search aspect types tool to test end to end functionality.",
				"authRequired": []string{"my-google-auth"},
			},
		},
	}

	return toolsFile
}

func runDataplexSearchEntriesToolInvokeTest(t *testing.T, tableName string, datasetName string) {
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	sessionId := tests.RunInitialize(t, "2024-11-05")

	testCases := []struct {
		name           string
		toolName       string
		requestHeader  map[string]string
		arguments      map[string]any
		wantStatusCode int
		expectResult   bool
		wantContentKey string
	}{
		{
			name:           "Success - Entry Found",
			toolName:       "my-dataplex-search-entries-tool",
			requestHeader:  map[string]string{},
			arguments:      map[string]any{"query": fmt.Sprintf("displayname=%s system=bigquery parent:%s", tableName, datasetName)},
			wantStatusCode: 200,
			expectResult:   true,
			wantContentKey: "dataplex_entry",
		},
		{
			name:           "Success with Authorization - Entry Found",
			toolName:       "my-auth-dataplex-search-entries-tool",
			requestHeader:  map[string]string{"my-google-auth_token": idToken},
			arguments:      map[string]any{"query": fmt.Sprintf("displayname=%s system=bigquery parent:%s", tableName, datasetName)},
			wantStatusCode: 200,
			expectResult:   true,
			wantContentKey: "dataplex_entry",
		},
		{
			name:           "Failure - Invalid Authorization Token",
			toolName:       "my-auth-dataplex-search-entries-tool",
			requestHeader:  map[string]string{"my-google-auth_token": "invalid_token"},
			arguments:      map[string]any{"query": fmt.Sprintf("displayname=%s system=bigquery parent:%s", tableName, datasetName)},
			wantStatusCode: 200, // MCP Protocol errors are always 200 OK
			expectResult:   false,
			wantContentKey: "dataplex_entry",
		},
		{
			name:           "Failure - Without Authorization Token",
			toolName:       "my-auth-dataplex-search-entries-tool",
			requestHeader:  map[string]string{},
			arguments:      map[string]any{"query": fmt.Sprintf("displayname=%s system=bigquery parent:%s", tableName, datasetName)},
			wantStatusCode: 200, // MCP errors are 200 OK
			expectResult:   false,
			wantContentKey: "dataplex_entry",
		},
		{
			name:           "Failure - Entry Not Found",
			toolName:       "my-dataplex-search-entries-tool",
			requestHeader:  map[string]string{},
			arguments:      map[string]any{"query": `displayname="" system=bigquery parent:""`},
			wantStatusCode: 200,
			expectResult:   false,
			wantContentKey: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mcpReq := map[string]any{
				"jsonrpc": "2.0",
				"id":      tc.toolName,
				"method":  "tools/call",
				"params": map[string]any{
					"name":      tc.toolName,
					"arguments": tc.arguments,
				},
			}
			reqBytes, _ := json.Marshal(mcpReq)

			req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:5000/mcp", bytes.NewBuffer(reqBytes))
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			if sessionId != "" {
				req.Header.Add("Mcp-Session-Id", sessionId)
			}
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

			var mcpResp struct {
				Error *struct {
					Message string `json:"message"`
				} `json:"error"`
				Result *struct {
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
					IsError bool `json:"isError"`
				} `json:"result"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
				t.Fatalf("Error parsing response body: %v", err)
			}

			if !tc.expectResult {
				if mcpResp.Error != nil {
					return
				}
				if mcpResp.Result != nil && mcpResp.Result.IsError {
					return
				}
				if mcpResp.Result != nil && len(mcpResp.Result.Content) > 0 {
					text := mcpResp.Result.Content[0].Text
					if text == "" || text == "[]" || text == "null" {
						return
					}
					t.Fatalf("expected an empty result or error message, but got: %s", text)
				}
				return // valid but empty result
			}

			if mcpResp.Error != nil {
				t.Fatalf("expected a result, but got MCP error: %s", mcpResp.Error.Message)
			}

			if mcpResp.Result == nil || len(mcpResp.Result.Content) == 0 {
				t.Fatalf("Expected a result with content, but it was empty")
			}

			resultStr := mcpResp.Result.Content[0].Text

			var entries []interface{}
			if err := json.Unmarshal([]byte(resultStr), &entries); err != nil {
				t.Fatalf("error unmarshalling result string: %v. Raw result: %s", err, resultStr)
			}

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
		})
	}
}

func runDataplexLookupEntryToolInvokeTest(t *testing.T, tableName string, datasetName string) {
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	sessionId := tests.RunInitialize(t, "2024-11-05")

	testCases := []struct {
		name               string
		wantStatusCode     int
		toolName           string
		requestHeader      map[string]string
		arguments          map[string]any
		expectResult       bool
		wantContentKey     string
		dontWantContentKey string
		aspectCheck        bool
	}{
		{
			name:           "Success - Entry Found",
			toolName:       "my-dataplex-lookup-entry-tool",
			requestHeader:  map[string]string{},
			arguments:      map[string]any{"name": fmt.Sprintf("projects/%s/locations/us", DataplexProject), "entry": fmt.Sprintf("projects/%s/locations/us/entryGroups/@bigquery/entries/bigquery.googleapis.com/projects/%s/datasets/%s", DataplexProject, DataplexProject, datasetName)},
			wantStatusCode: 200,
			expectResult:   true,
			wantContentKey: "name",
		},
		{
			name:           "Success - Entry Found with Authorization",
			toolName:       "my-auth-dataplex-lookup-entry-tool",
			requestHeader:  map[string]string{"my-google-auth_token": idToken},
			arguments:      map[string]any{"name": fmt.Sprintf("projects/%s/locations/us", DataplexProject), "entry": fmt.Sprintf("projects/%s/locations/us/entryGroups/@bigquery/entries/bigquery.googleapis.com/projects/%s/datasets/%s", DataplexProject, DataplexProject, datasetName)},
			wantStatusCode: 200,
			expectResult:   true,
			wantContentKey: "name",
		},
		{
			name:           "Failure - Invalid Authorization Token",
			toolName:       "my-auth-dataplex-lookup-entry-tool",
			requestHeader:  map[string]string{"my-google-auth_token": "invalid_token"},
			arguments:      map[string]any{"name": fmt.Sprintf("projects/%s/locations/us", DataplexProject), "entry": fmt.Sprintf("projects/%s/locations/us/entryGroups/@bigquery/entries/bigquery.googleapis.com/projects/%s/datasets/%s", DataplexProject, DataplexProject, datasetName)},
			wantStatusCode: 200,
			expectResult:   false,
			wantContentKey: "name",
		},
		{
			name:           "Failure - Without Authorization Token",
			toolName:       "my-auth-dataplex-lookup-entry-tool",
			requestHeader:  map[string]string{},
			arguments:      map[string]any{"name": fmt.Sprintf("projects/%s/locations/us", DataplexProject), "entry": fmt.Sprintf("projects/%s/locations/us/entryGroups/@bigquery/entries/bigquery.googleapis.com/projects/%s/datasets/%s", DataplexProject, DataplexProject, datasetName)},
			wantStatusCode: 200,
			expectResult:   false,
			wantContentKey: "name",
		},
		{
			name:           "Failure - Entry Not Found or Permission Denied",
			toolName:       "my-dataplex-lookup-entry-tool",
			requestHeader:  map[string]string{},
			arguments:      map[string]any{"name": fmt.Sprintf("projects/%s/locations/us", DataplexProject), "entry": fmt.Sprintf("projects/%s/locations/us/entryGroups/@bigquery/entries/bigquery.googleapis.com/projects/%s/datasets/%s", DataplexProject, DataplexProject, "non-existent-dataset")},
			wantStatusCode: 200,
			expectResult:   false,
		},
		{
			name:               "Success - Entry Found with Basic View",
			toolName:           "my-dataplex-lookup-entry-tool",
			requestHeader:      map[string]string{},
			arguments:          map[string]any{"name": fmt.Sprintf("projects/%s/locations/us", DataplexProject), "entry": fmt.Sprintf("projects/%s/locations/us/entryGroups/@bigquery/entries/bigquery.googleapis.com/projects/%s/datasets/%s/tables/%s", DataplexProject, DataplexProject, datasetName, tableName), "view": 1},
			wantStatusCode:     200,
			expectResult:       true,
			wantContentKey:     "name",
			dontWantContentKey: "aspects",
		},
		{
			name:           "Failure - Entry with Custom View without Aspect Types",
			toolName:       "my-dataplex-lookup-entry-tool",
			requestHeader:  map[string]string{},
			arguments:      map[string]any{"name": fmt.Sprintf("projects/%s/locations/us", DataplexProject), "entry": fmt.Sprintf("projects/%s/locations/us/entryGroups/@bigquery/entries/bigquery.googleapis.com/projects/%s/datasets/%s/tables/%s", DataplexProject, DataplexProject, datasetName, tableName), "view": 3},
			wantStatusCode: 200,
			expectResult:   false,
		},
		{
			name:           "Success - Entry Found with only Schema Aspect",
			toolName:       "my-dataplex-lookup-entry-tool",
			requestHeader:  map[string]string{},
			arguments:      map[string]any{"name": fmt.Sprintf("projects/%s/locations/us", DataplexProject), "entry": fmt.Sprintf("projects/%s/locations/us/entryGroups/@bigquery/entries/bigquery.googleapis.com/projects/%s/datasets/%s/tables/%s", DataplexProject, DataplexProject, datasetName, tableName), "aspectTypes": []string{"projects/dataplex-types/locations/global/aspectTypes/schema"}, "view": 3},
			wantStatusCode: 200,
			expectResult:   true,
			wantContentKey: "aspects",
			aspectCheck:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mcpReq := map[string]any{
				"jsonrpc": "2.0",
				"id":      tc.toolName,
				"method":  "tools/call",
				"params": map[string]any{
					"name":      tc.toolName,
					"arguments": tc.arguments,
				},
			}
			reqBytes, _ := json.Marshal(mcpReq)

			req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:5000/mcp", bytes.NewBuffer(reqBytes))
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			if sessionId != "" {
				req.Header.Add("Mcp-Session-Id", sessionId)
			}
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

			var mcpResp struct {
				Error *struct {
					Message string `json:"message"`
				} `json:"error"`
				Result *struct {
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
					IsError bool `json:"isError"`
				} `json:"result"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
				t.Fatalf("Error parsing response body: %v", err)
			}

			if !tc.expectResult {
				if mcpResp.Error != nil {
					return
				}
				if mcpResp.Result != nil && mcpResp.Result.IsError {
					return
				}
				if mcpResp.Result != nil && len(mcpResp.Result.Content) > 0 {
					text := mcpResp.Result.Content[0].Text
					if text == "" || text == "{}" || text == "null" {
						return
					}
					t.Fatalf("Expected an error in response, but none was found. Raw result: %s", text)
				}
				return
			}

			if mcpResp.Error != nil {
				t.Fatalf("expected a result, but got MCP error: %s", mcpResp.Error.Message)
			}

			if mcpResp.Result == nil || len(mcpResp.Result.Content) == 0 {
				t.Fatalf("Expected a result with content, but it was empty")
			}

			resultStr := mcpResp.Result.Content[0].Text

			var entry map[string]interface{}
			if err := json.Unmarshal([]byte(resultStr), &entry); err != nil {
				t.Fatalf("Error unmarshalling result string: %v. Raw result: %s", err, resultStr)
			}

			if _, ok := entry[tc.wantContentKey]; !ok {
				t.Fatalf("Expected entry to have key '%s', but it was not found in %v", tc.wantContentKey, entry)
			}

			if tc.dontWantContentKey != "" {
				if _, ok := entry[tc.dontWantContentKey]; ok {
					t.Fatalf("Expected entry to NOT have key '%s', but it was found", tc.dontWantContentKey)
				}
			}

			if tc.aspectCheck {
				aspects, ok := entry["aspects"].(map[string]interface{})
				if !ok || len(aspects) != 1 {
					t.Fatalf("Expected exactly one aspect, but got %d", len(aspects))
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

	sessionId := tests.RunInitialize(t, "2024-11-05")

	testCases := []struct {
		name           string
		toolName       string
		requestHeader  map[string]string
		arguments      map[string]any
		wantStatusCode int
		expectResult   bool
		wantContentKey string
	}{
		{
			name:           "Success - Aspect Type Found",
			toolName:       "my-dataplex-search-aspect-types-tool",
			requestHeader:  map[string]string{},
			arguments:      map[string]any{"query": fmt.Sprintf("name:%s_aspectType", aspectTypeId)},
			wantStatusCode: 200,
			expectResult:   true,
			wantContentKey: "metadata_template",
		},
		{
			name:           "Success - Aspect Type Found with Authorization",
			toolName:       "my-auth-dataplex-search-aspect-types-tool",
			requestHeader:  map[string]string{"my-google-auth_token": idToken},
			arguments:      map[string]any{"query": fmt.Sprintf("name:%s_aspectType", aspectTypeId)},
			wantStatusCode: 200,
			expectResult:   true,
			wantContentKey: "metadata_template",
		},
		{
			name:           "Failure - Aspect Type Not Found",
			toolName:       "my-dataplex-search-aspect-types-tool",
			requestHeader:  map[string]string{},
			arguments:      map[string]any{"query": "name:_aspectType"},
			wantStatusCode: 200,
			expectResult:   false,
		},
		{
			name:           "Failure - Invalid Authorization Token",
			toolName:       "my-auth-dataplex-search-aspect-types-tool",
			requestHeader:  map[string]string{"my-google-auth_token": "invalid_token"},
			arguments:      map[string]any{"query": fmt.Sprintf("name:%s_aspectType", aspectTypeId)},
			wantStatusCode: 200, // MCP Protocol errors are 200 OK
			expectResult:   false,
		},
		{
			name:           "Failure - No Authorization Token",
			toolName:       "my-auth-dataplex-search-aspect-types-tool",
			requestHeader:  map[string]string{},
			arguments:      map[string]any{"query": fmt.Sprintf("name:%s_aspectType", aspectTypeId)},
			wantStatusCode: 200,
			expectResult:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mcpReq := map[string]any{
				"jsonrpc": "2.0",
				"id":      tc.toolName,
				"method":  "tools/call",
				"params": map[string]any{
					"name":      tc.toolName,
					"arguments": tc.arguments,
				},
			}
			reqBytes, _ := json.Marshal(mcpReq)

			req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:5000/mcp", bytes.NewBuffer(reqBytes))
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			if sessionId != "" {
				req.Header.Add("Mcp-Session-Id", sessionId)
			}
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

			var mcpResp struct {
				Error *struct {
					Message string `json:"message"`
				} `json:"error"`
				Result *struct {
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
					IsError bool `json:"isError"`
				} `json:"result"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
				t.Fatalf("Error parsing response body: %v", err)
			}

			if !tc.expectResult {
				if mcpResp.Error != nil {
					return
				}
				if mcpResp.Result != nil && mcpResp.Result.IsError {
					return
				}
				if mcpResp.Result != nil && len(mcpResp.Result.Content) > 0 {
					text := mcpResp.Result.Content[0].Text
					if text == "" || text == "[]" || text == "null" {
						return
					}
					t.Fatalf("expected an empty result or error message, but got: %s", text)
				}
				return // valid but empty result
			}

			if mcpResp.Error != nil {
				t.Fatalf("expected a result, but got MCP error: %s", mcpResp.Error.Message)
			}

			if mcpResp.Result == nil || len(mcpResp.Result.Content) == 0 {
				t.Fatalf("Expected a result with content, but it was empty")
			}

			resultStr := mcpResp.Result.Content[0].Text

			var entries []interface{}
			if err := json.Unmarshal([]byte(resultStr), &entries); err != nil {
				t.Fatalf("error unmarshalling result string: %v. Raw result: %s", err, resultStr)
			}

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
		})
	}
}
