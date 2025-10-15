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

package oracle_test

import (
        "bytes"
        "context"
        "encoding/json"
        "fmt"
        "io"
        "net/http"
        "testing"
        "time"

        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
)

// TestOracleSQLParameterized tests the oracle-sql tool with pre-defined parameterized queries
func TestOracleSQLParameterized(t *testing.T) {
        if !isMCPServerRunning() {
                t.Skip("MCP server is not running at " + mcpServerURL)
        }

        ctx := context.Background()
        tableName := "test_oracle_users"

        // Setup test data
        t.Run("00_Setup", func(t *testing.T) {
                setupParameterizedTestData(t, ctx, tableName)
        })

        defer func() {
                t.Log("Cleaning up test table...")
                cleanupTestTable(ctx, tableName)
        }()

        // Test parameterized queries
        t.Run("01_SearchByName", func(t *testing.T) {
                testParamSearchByName(t, ctx)
        })

        t.Run("02_SearchByAge", func(t *testing.T) {
                testParamSearchByAge(t, ctx)
        })

        t.Run("03_SearchByDeptAndAge", func(t *testing.T) {
                testParamSearchByDeptAndAge(t, ctx)
        })

        t.Run("04_GetUserById", func(t *testing.T) {
                testParamGetUserById(t, ctx)
        })

        t.Run("05_CountUsersByDept", func(t *testing.T) {
                testParamCountUsersByDept(t, ctx)
        })

        t.Run("06_GetUsersInDept_MultiRow", func(t *testing.T) {
                testParamGetUsersInDept(t, ctx)
        })

        t.Run("07_SearchWithNull", func(t *testing.T) {
                testParamSearchWithNull(t, ctx)
        })

        t.Run("08_ComplexQuery", func(t *testing.T) {
                testParamComplexQuery(t, ctx)
        })
}

func setupParameterizedTestData(t *testing.T, ctx context.Context, tableName string) {
        // Create table
        createSQL := fmt.Sprintf(`
                CREATE TABLE %s (
                        id NUMBER PRIMARY KEY,
                        name VARCHAR2(100),
                        email VARCHAR2(100),
                        age NUMBER,
                        department VARCHAR2(50)
                )
        `, tableName)

        _, err := executeMCPTool(ctx, createSQL)
        require.NoError(t, err, "Failed to create test table")

        // Insert test data
        testData := []struct {
                id         int
                name       string
                email      string
                age        int
                department string
        }{
                {1, "Alice Smith", "alice@example.com", 25, "Engineering"},
                {2, "Bob Johnson", "bob@example.com", 30, "Engineering"},
                {3, "Charlie Brown", "charlie@example.com", 28, "Sales"},
                {4, "David Wilson", "david@example.com", 35, "Engineering"},
                {5, "Eve Davis", "eve@example.com", 22, "Marketing"},
                {6, "Frank Miller", "frank@example.com", 27, "Engineering"},
                {7, "Grace Lee", "", 29, "Sales"}, // NULL email for testing
        }

        for _, data := range testData {
                var insertSQL string
                if data.email == "" {
                        insertSQL = fmt.Sprintf(`
                                INSERT INTO %s (id, name, email, age, department)
                                VALUES (%d, '%s', NULL, %d, '%s')
                        `, tableName, data.id, data.name, data.age, data.department)
                } else {
                        insertSQL = fmt.Sprintf(`
                                INSERT INTO %s (id, name, email, age, department)
                                VALUES (%d, '%s', '%s', %d, '%s')
                        `, tableName, data.id, data.name, data.email, data.age, data.department)
                }

                _, err := executeMCPTool(ctx, insertSQL)
                require.NoError(t, err, "Failed to insert test data")
        }

        t.Logf("✅ Created test table with 7 rows: %s", tableName)
}

func testParamSearchByName(t *testing.T, ctx context.Context) {
        result, err := callMCPToolWithParams(ctx, "test-search-users-by-name", map[string]interface{}{
                "search_name": "Smith",
        })

        require.NoError(t, err, "Failed to search by name")

        rows, ok := result.([]interface{})
        require.True(t, ok, "Expected array result")
        assert.GreaterOrEqual(t, len(rows), 1, "Should find at least 1 user with 'Smith'")

        if len(rows) > 0 {
                row := rows[0].(map[string]interface{})
                name := fmt.Sprintf("%v", row["NAME"])
                assert.Contains(t, name, "Smith", "Name should contain 'Smith'")
        }

        t.Logf("✅ Search by name found %d rows", len(rows))
}

func testParamSearchByAge(t *testing.T, ctx context.Context) {
        result, err := callMCPToolWithParams(ctx, "test-search-users-by-age", map[string]interface{}{
                "min_age": 25,
                "max_age": 30,
        })

        require.NoError(t, err, "Failed to search by age")

        rows, ok := result.([]interface{})
        require.True(t, ok, "Expected array result")
        assert.GreaterOrEqual(t, len(rows), 3, "Should find at least 3 users in age range 25-30")

        for _, rowInterface := range rows {
                row := rowInterface.(map[string]interface{})
                ageStr := fmt.Sprintf("%v", row["AGE"])
                t.Logf("   Found user age: %s", ageStr)
        }

        t.Logf("✅ Search by age found %d rows", len(rows))
}

func testParamSearchByDeptAndAge(t *testing.T, ctx context.Context) {
        result, err := callMCPToolWithParams(ctx, "test-search-users-by-dept-and-age", map[string]interface{}{
                "department": "Engineering",
                "min_age":    26,
        })

        require.NoError(t, err, "Failed to search by dept and age")

        rows, ok := result.([]interface{})
        require.True(t, ok, "Expected array result")
        assert.GreaterOrEqual(t, len(rows), 2, "Should find at least 2 engineers over 26")

        for _, rowInterface := range rows {
                row := rowInterface.(map[string]interface{})
                dept := fmt.Sprintf("%v", row["DEPARTMENT"])
                assert.Contains(t, dept, "Engineering", "Department should be Engineering")
        }

        t.Logf("✅ Multi-parameter search found %d rows", len(rows))
}

func testParamGetUserById(t *testing.T, ctx context.Context) {
        result, err := callMCPToolWithParams(ctx, "test-get-user-by-id", map[string]interface{}{
                "user_id": 1,
        })

        require.NoError(t, err, "Failed to get user by ID")

        rows, ok := result.([]interface{})
        require.True(t, ok, "Expected array result")
        require.Len(t, rows, 1, "Should find exactly 1 user with ID 1")

        row := rows[0].(map[string]interface{})
        assert.Equal(t, "1", fmt.Sprintf("%v", row["ID"]))
        assert.Contains(t, fmt.Sprintf("%v", row["NAME"]), "Alice")

        t.Logf("✅ Get user by ID successful")
        t.Logf("   User: %v", row)
}

func testParamCountUsersByDept(t *testing.T, ctx context.Context) {
        result, err := callMCPToolWithParams(ctx, "test-count-users-by-dept", map[string]interface{}{
                "department": "Engineering",
        })

        require.NoError(t, err, "Failed to count users by dept")

        rows, ok := result.([]interface{})
        require.True(t, ok, "Expected array result")
        require.Len(t, rows, 1, "Should return 1 row with count")

        row := rows[0].(map[string]interface{})
        count := fmt.Sprintf("%v", row["USER_COUNT"])
        assert.Equal(t, "4", count, "Should have 4 users in Engineering")

        t.Logf("✅ Count users by dept: %s users in Engineering", count)
}

func testParamGetUsersInDept(t *testing.T, ctx context.Context) {
        result, err := callMCPToolWithParams(ctx, "test-get-users-in-dept", map[string]interface{}{
                "department": "Engineering",
        })

        require.NoError(t, err, "Failed to get users in dept")

        // CRITICAL: Verify multi-row result is array
        rows, ok := result.([]interface{})
        require.True(t, ok, "Expected array result for multi-row query")
        assert.Equal(t, 4, len(rows), "Should find exactly 4 Engineering users")

        t.Logf("✅ Get users in dept returned %d rows (multi-row test passed!)", len(rows))
        for i, rowInterface := range rows {
                row := rowInterface.(map[string]interface{})
                t.Logf("   User %d: %v (%v)", i+1, row["NAME"], row["AGE"])
        }
}

func testParamSearchWithNull(t *testing.T, ctx context.Context) {
        result, err := callMCPToolWithParams(ctx, "test-search-with-null", map[string]interface{}{})

        require.NoError(t, err, "Failed to search with NULL")

        rows, ok := result.([]interface{})
        require.True(t, ok, "Expected array result")
        assert.GreaterOrEqual(t, len(rows), 1, "Should find at least 1 user with NULL email")

        if len(rows) > 0 {
                row := rows[0].(map[string]interface{})
                email := row["EMAIL"]
                assert.True(t, email == nil || email == "", "Email should be NULL or empty")
        }

        t.Logf("✅ NULL handling test found %d rows", len(rows))
}

func testParamComplexQuery(t *testing.T, ctx context.Context) {
        result, err := callMCPToolWithParams(ctx, "test-complex-query", map[string]interface{}{
                "min_age": 25,
        })

        require.NoError(t, err, "Failed to execute complex query")

        rows, ok := result.([]interface{})
        require.True(t, ok, "Expected array result")
        assert.GreaterOrEqual(t, len(rows), 4, "Should find at least 4 users >= 25")

        // Verify computed columns exist
        row := rows[0].(map[string]interface{})
        assert.Contains(t, row, "SENIORITY", "Should have SENIORITY column")
        assert.Contains(t, row, "MONTHS_OLD", "Should have MONTHS_OLD column")

        t.Logf("✅ Complex query successful: %d rows", len(rows))
        t.Logf("   Sample: %v (Seniority: %v, Months: %v)", 
                row["NAME"], row["SENIORITY"], row["MONTHS_OLD"])
}

// Helper function to call MCP tool with parameters
func callMCPToolWithParams(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
        request := MCPRequest{
                JSONRPC: "2.0",
                ID:      1,
                Method:  "tools/call",
                Params: map[string]interface{}{
                        "name":      toolName,
                        "arguments": params,
                },
        }

        jsonData, err := json.Marshal(request)
        if err != nil {
                return nil, fmt.Errorf("failed to marshal request: %w", err)
        }

        httpReq, err := http.NewRequestWithContext(ctx, "POST", mcpServerURL, bytes.NewBuffer(jsonData))
        if err != nil {
                return nil, fmt.Errorf("failed to create request: %w", err)
        }
        httpReq.Header.Set("Content-Type", "application/json")

        client := &http.Client{Timeout: 30 * time.Second}
        resp, err := client.Do(httpReq)
        if err != nil {
                return nil, fmt.Errorf("failed to execute request: %w", err)
        }
        defer resp.Body.Close()

        body, err := io.ReadAll(resp.Body)
        if err != nil {
                return nil, fmt.Errorf("failed to read response: %w", err)
        }

        var mcpResp MCPResponse
        if err := json.Unmarshal(body, &mcpResp); err != nil {
                return nil, fmt.Errorf("failed to unmarshal response: %w", err)
        }

        if mcpResp.Error != nil {
                return nil, fmt.Errorf("MCP error: %v", mcpResp.Error)
        }

        content, ok := mcpResp.Result["content"].([]interface{})
        if !ok {
                return nil, fmt.Errorf("invalid content format in response")
        }

        var results []interface{}

        if len(content) == 0 {
                return []interface{}{}, nil
        }

        for _, item := range content {
                contentItem, ok := item.(map[string]interface{})
                if !ok {
                        continue
                }

                text, ok := contentItem["text"].(string)
                if !ok || text == "" {
                        continue
                }

                var rowData interface{}
                if err := json.Unmarshal([]byte(text), &rowData); err != nil {
                        results = append(results, text)
                        continue
                }

                results = append(results, rowData)
        }

        return results, nil
}
