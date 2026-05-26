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

package cloudsqlpg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"
	"github.com/jackc/pgx/v5/pgxpool"
)

func createPostgresExtension(t *testing.T, ctx context.Context, pool *pgxpool.Pool, extensionName string) func() {
	createExtensionCmd := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", extensionName)
	_, err := pool.Exec(ctx, createExtensionCmd)
	if err != nil {
		t.Fatalf("failed to create extension: %v", err)
	}
	return func() {
		dropExtensionCmd := fmt.Sprintf("DROP EXTENSION IF EXISTS %s", extensionName)
		_, err := pool.Exec(ctx, dropExtensionCmd)
		if err != nil {
			t.Fatalf("failed to drop extension: %v", err)
		}
	}
}

// setupVectorAssistTable prepares the database extensions and test data needed
// to test the definespec, modifyspec, applyspec, and generatequery tools.
func setupVectorAssistTable(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (string, func(t *testing.T), func()) {
	// Install necessary extensions for VectorAssist
	dropExtensionFunc := createPostgresExtension(t, ctx, pool, "vector_assist")

	uniqueID := strings.ReplaceAll(uuid.New().String(), "-", "")
	tableName := "vector_assist_test_" + uniqueID

	// Create a table with vector data for defining/modifying/applying specs
	createStmt := fmt.Sprintf(`
		CREATE TABLE %s (
			name TEXT,
			category TEXT,
			content TEXT,
			embedding vector(3)
		);
	`, tableName)

	_, err := pool.Exec(ctx, createStmt)
	if err != nil {
		t.Fatalf("failed to create vector assist test table: %v", err)
	}

	// Insert sample data to generate queries against
	insertDataStmt := fmt.Sprintf(`
		INSERT INTO %s (name, category, content, embedding)
		VALUES
		('Item 1', 'Document', 'Sample text document about AI', array_fill(0.1, ARRAY[3])::vector),
		('Item 2', 'Document', 'Sample text document about databases', array_fill(0.2, ARRAY[3])::vector);
	`, tableName)

	_, err = pool.Exec(ctx, insertDataStmt)
	if err != nil {
		t.Fatalf("failed to insert data into vector assist table: %v", err)
	}

	// Return teardown function
	teardown := func(t *testing.T) {
		_, err := pool.Exec(context.Background(), fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName))
		if err != nil {
			t.Errorf("failed to drop vector assist table %s: %v", tableName, err)
		}
	}

	return tableName, teardown, dropExtensionFunc
}

// TODO: Remove the test from this file and follow the existing test pattern
// by calling the tests from cloudsqlpg_integration_test.go
func TestVectorAssistIntegration(t *testing.T) {
	sourceConfig := getCloudSQLPgVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	args := []string{"--enable-api"}

	pool, err := initCloudSQLPgConnectionPool(CloudSQLPostgresProject, CloudSQLPostgresRegion, CloudSQLPostgresInstance, "public", CloudSQLPostgresUser, CloudSQLPostgresPass, CloudSQLPostgresDatabase)
	if err != nil {
		t.Fatalf("unable to create Cloud SQL connection pool: %s", err)
	}

	// Generate a unique ID
	uniqueID := strings.ReplaceAll(uuid.New().String(), "-", "")

	// This will execute after all tool tests complete (success, fail, or t.Fatal)
	t.Cleanup(func() {
		tests.CleanupPostgresTables(t, context.Background(), pool, uniqueID)
	})

	//Create table names using the UUID
	tableNameParam := "param_table_" + uniqueID
	tableNameAuth := "auth_table_" + uniqueID

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := tests.GetPostgresSQLParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupPostgresSQLTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetPostgresSQLAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupPostgresSQLTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// // Set up data for vector assist tools
	vectorAssistTableName, teardownVectorAssistTable, dropExtension := setupVectorAssistTable(t, ctx, pool)
	defer teardownVectorAssistTable(t)
	defer dropExtension()

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, CloudSQLPostgresToolType, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddExecuteSqlConfig(t, toolsFile, "postgres-execute-sql")
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetPostgresSQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, CloudSQLPostgresToolType, tmplSelectCombined, tmplSelectFilterCombined, "")

	// Add vector assist tools to the configuration
	toolsFile = AddVectorAssistConfig(t, toolsFile, "my-instance")

	toolsFile = tests.AddPostgresPrebuiltConfig(t, toolsFile)
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

	// Run vectorassist tool tests
	specID := "va_spec_001"
	RunVectorAssistDefineSpecToolInvokeTest(t, ctx, pool, vectorAssistTableName, specID)
	RunVectorAssistModifySpecToolInvokeTest(t, ctx, pool, specID)
	RunVectorAssistApplySpecToolInvokeTest(t, ctx, pool, specID)
	RunVectorAssistGenerateQueryToolInvokeTest(t, ctx, pool, specID)
}

// AddVectorAssistConfig appends the vector assist tool configurations to the given tools file.
func AddVectorAssistConfig(t *testing.T, config map[string]any, sourceName string) map[string]any {
	tools, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}

	tools["define_spec"] = map[string]any{
		"type":   "vector-assist-define-spec",
		"source": sourceName,
	}
	tools["modify_spec"] = map[string]any{
		"type":   "vector-assist-modify-spec",
		"source": sourceName,
	}
	tools["apply_spec"] = map[string]any{
		"type":   "vector-assist-apply-spec",
		"source": sourceName,
	}
	tools["generate_query"] = map[string]any{
		"type":   "vector-assist-generate-query",
		"source": sourceName,
	}
	config["tools"] = tools
	return config
}

func RunVectorAssistDefineSpecToolInvokeTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tableName string, specID string) {
	validPayload := fmt.Sprintf(`{
		"table_name": "%s",
		"schema_name": "public",
		"spec_id": "%s",
		"vector_column_name": "embedding",
		"text_column_name": "content",
		"vector_index_type": "hnsw",
		"embeddings_available": true,
		"num_vectors": 2,
		"dimensionality": 3,
		"embedding_model": "textembedding-gecko",
		"prefilter_column_names": ["category"],
		"distance_func": "cosine",
		"quantization": "halfvec",
		"memory_budget_kb": 1024,
		"target_recall": 0.95,
		"target_top_k": 10,
		"tune_vector_index": true
	}`, tableName, specID)

	tcs := []struct {
		name           string
		requestBody    io.Reader
		api            string
		wantStatusCode int
		wantContains   []string
	}{
		{
			name:           "invoke define_spec with all valid parameters",
			requestBody:    bytes.NewBuffer([]byte(validPayload)),
			api:            "http://127.0.0.1:5000/api/tool/define_spec/invoke",
			wantStatusCode: http.StatusOK,
			// Check for key identifiers instead of the entire JSON string
			wantContains: []string{
				`"vector_spec_id":"va_spec_001"`,
				fmt.Sprintf(`"table_name":"%s"`, tableName),
				`"recommendation_id"`, // Ensure a recommendation was generated
			},
		},
		{
			name:           "invoke define_spec with missing required table_name",
			requestBody:    bytes.NewBuffer([]byte(`{"schema_name": "public", "spec_id": "va_spec_002"}`)),
			api:            "http://127.0.0.1:5000/api/tool/define_spec/invoke",
			wantStatusCode: http.StatusOK,
			wantContains: []string{
				`"error"`,
			},
		},
	}
	for _, tc := range tcs {
		runVectorAssistToolInvokeTest(t, tc)
	}
}

func RunVectorAssistModifySpecToolInvokeTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool, specID string) {
	validPayload := fmt.Sprintf(`{
        "spec_id": "%s",
        "memory_budget_kb": 2048,
        "target_recall": 0.99
    }`, specID)

	tcs := []struct {
		name           string
		requestBody    io.Reader
		api            string
		wantStatusCode int
		wantContains   []string
	}{
		{
			name:           "modify existing spec with new constraints",
			requestBody:    bytes.NewBuffer([]byte(validPayload)),
			api:            "http://127.0.0.1:5000/api/tool/modify_spec/invoke",
			wantStatusCode: http.StatusOK,
			wantContains: []string{
				`"recommendation_id"`,
			},
		},
		{
			name:           "modify existing spec without required spec id",
			requestBody:    bytes.NewBuffer([]byte(`{"target_recall": 0.99}`)),
			api:            "http://127.0.0.1:5000/api/tool/modify_spec/invoke",
			wantStatusCode: http.StatusOK,
			wantContains: []string{
				`"error"`,
			},
		},
	}
	for _, tc := range tcs {
		runVectorAssistToolInvokeTest(t, tc)
	}
}

func RunVectorAssistApplySpecToolInvokeTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool, recommendationID string) {
	validPayload := fmt.Sprintf(`{
        "spec_id": "%s"
    }`, recommendationID)

	tcs := []struct {
		name           string
		requestBody    io.Reader
		api            string
		wantStatusCode int
		wantContains   []string
	}{
		{
			name:           "apply recommendation to database",
			requestBody:    bytes.NewBuffer([]byte(validPayload)),
			api:            "http://127.0.0.1:5000/api/tool/apply_spec/invoke",
			wantStatusCode: http.StatusOK,
			wantContains: []string{
				`{"apply_spec":true}`,
			},
		},
		{
			name:           "apply recommendation to database without spec id",
			requestBody:    bytes.NewBuffer([]byte(`{"schema_name": "public"}`)),
			api:            "http://127.0.0.1:5000/api/tool/apply_spec/invoke",
			wantStatusCode: http.StatusOK,
			wantContains: []string{
				`"error"`,
			},
		},
	}

	for _, tc := range tcs {
		runVectorAssistToolInvokeTest(t, tc)
	}
}

func RunVectorAssistGenerateQueryToolInvokeTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool, specID string) {
	validPayload := fmt.Sprintf(`{
        "spec_id": "%s",
        "search_text": "What is the capital of France?",
        "top_k": 5
    }`, specID)

	tcs := []struct {
		name           string
		requestBody    io.Reader
		api            string
		wantStatusCode int
		wantContains   []string
	}{
		{
			name:           "generate SQL for vector search",
			requestBody:    bytes.NewBuffer([]byte(validPayload)),
			api:            "http://127.0.0.1:5000/api/tool/generate_query/invoke",
			wantStatusCode: http.StatusOK,
			wantContains: []string{
				`"generate_query"`,
				`LIMIT 5`,
			},
		},
	}

	for _, tc := range tcs {
		runVectorAssistToolInvokeTest(t, tc)
	}
}

func runVectorAssistToolInvokeTest(t *testing.T, tc struct {
	name           string
	requestBody    io.Reader
	api            string
	wantStatusCode int
	wantContains   []string
}) {
	t.Run(tc.name, func(t *testing.T) {
		resp, body := tests.RunRequest(t, http.MethodPost, tc.api, tc.requestBody, nil)

		if resp.StatusCode != tc.wantStatusCode {
			t.Fatalf("tool %s: wrong status code: got %d, want %d, body: %s", tc.api, resp.StatusCode, tc.wantStatusCode, string(body))
		}

		if tc.wantStatusCode != http.StatusOK {
			return
		}

		// Unmarshal the standard response wrapper
		var bodyWrapper struct {
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(body, &bodyWrapper); err != nil {
			t.Fatalf("error decoding response wrapper: %v", err)
		}

		// Handle the double-unmarshal logic for stringified results
		var resultString string
		if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
			resultString = string(bodyWrapper.Result)
		}

		// Verification loop
		for _, expectedSubstr := range tc.wantContains {
			if !strings.Contains(resultString, expectedSubstr) {
				t.Errorf("Expected result to contain %q, but it did not.\nFull result: %s", expectedSubstr, resultString)
			}
		}
	})
}
