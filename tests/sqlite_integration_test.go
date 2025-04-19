//go:build integration && sqlite

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

package tests

import (
    "context"
    "database/sql"
    "fmt"
    "os"
    "regexp"
    "strings"
    "testing"
    "time"

    "github.com/google/uuid"
)

var (
    SQLITE_SOURCE_KIND = "sqlite"
    SQLITE_TOOL_KIND   = "sqlite-sql"
    SQLITE_DATABASE    = os.Getenv("SQLITE_DATABASE")
)

func getSQLiteVars(t *testing.T) map[string]any {
    return map[string]any{
        "kind":     SQLITE_SOURCE_KIND,
        "database": SQLITE_DATABASE,
    }
}

// SetupSQLiteTestDB creates a temporary SQLite database for testing
func SetupSQLiteTestDB(t *testing.T) (func(t *testing.T), error) {
    if SQLITE_DATABASE == "" {
        // Create a temporary database file
        tmpFile, err := os.CreateTemp("", "test-*.db")
        if err != nil {
            return nil, fmt.Errorf("failed to create temp file: %v", err)
        }
        SQLITE_DATABASE = tmpFile.Name()
    }

    // Open database connection
    db, err := sql.Open("sqlite", SQLITE_DATABASE)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %v", err)
    }
    defer db.Close()

    // Create test table
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS test_table (
            id INTEGER PRIMARY KEY,
            name TEXT NOT NULL,
            value INTEGER,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
    if err != nil {
        return nil, fmt.Errorf("failed to create test table: %v", err)
    }

    cleanup := func(t *testing.T) {
        if err := os.Remove(SQLITE_DATABASE); err != nil {
            t.Logf("Failed to remove test database: %v", err)
        }
    }

    return cleanup, nil
}

func TestSQLiteConnection(t *testing.T) {
    cleanup, err := SetupSQLiteTestDB(t)
    if err != nil {
        t.Fatal(err)
    }
    defer cleanup(t)

    err = RunSourceConnectionTest(t, getSQLiteVars(t), SQLITE_TOOL_KIND)
    if err != nil {
        t.Fatalf("Connection test failure: %s", err)
    }
}

func TestSQLiteToolEndpoint(t *testing.T) {
    cleanup, err := SetupSQLiteTestDB(t)
    if err != nil {
        t.Fatal(err)
    }
    defer cleanup(t)

    testID := uuid.New().String()
    timestamp := time.Now().UTC()

    testCases := []struct {
        name     string
        config   map[string]any
        validate func(t *testing.T, res []any)
    }{
        {
            name: "insert and select",
            config: map[string]any{
                "kind":        SQLITE_TOOL_KIND,
                "name":        "test-sqlite-insert",
                "source":      "test-sqlite",
                "description": "Test SQLite insert",
                "statement": `INSERT INTO test_table (name, value) VALUES (?, ?) 
                            RETURNING id, name, value, created_at`,
                "parameters": []map[string]any{
                    {
                        "name":        "name",
                        "type":        "string",
                        "description": "Name to insert",
                    },
                    {
                        "name":        "value",
                        "type":        "integer",
                        "description": "Value to insert",
                    },
                },
            },
            validate: func(t *testing.T, res []any) {
                if len(res) != 1 {
                    t.Fatalf("expected 1 result, got %d", len(res))
                }
                row := res[0].(map[string]any)
                if row["name"] != testID {
                    t.Errorf("expected name %s, got %s", testID, row["name"])
                }
                if row["value"] != int64(42) {
                    t.Errorf("expected value 42, got %v", row["value"])
                }
            },
        },
    }

    sourceConfig := map[string]any{
        "kind":     SQLITE_SOURCE_KIND,
        "name":     "test-sqlite",
        "database": SQLITE_DATABASE,
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            res, err := RunTest(t, sourceConfig, tc.config, map[string]any{
                "name":  testID,
                "value": 42,
            })
            if err != nil {
                t.Fatalf("Test failure: %s", err)
            }
            tc.validate(t, res)
        })
    }
}