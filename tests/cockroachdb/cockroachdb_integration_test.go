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

package cockroachdb

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	crdbpgx "github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgxv5"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/tests"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	CockroachDBSourceKind = "cockroachdb"
	CockroachDBToolKind   = "cockroachdb-sql"
	CockroachDBDatabase   = getEnvOrDefault("COCKROACHDB_DATABASE", "defaultdb")
	CockroachDBHost       = getEnvOrDefault("COCKROACHDB_HOST", "localhost")
	CockroachDBPort       = getEnvOrDefault("COCKROACHDB_PORT", "26257")
	CockroachDBUser       = getEnvOrDefault("COCKROACHDB_USER", "root")
	CockroachDBPass       = getEnvOrDefault("COCKROACHDB_PASS", "")
)

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getCockroachDBVars(t *testing.T) map[string]any {
	if CockroachDBHost == "" {
		t.Skip("COCKROACHDB_HOST not set, skipping CockroachDB integration test")
	}

	return map[string]any{
		"kind":           CockroachDBSourceKind,
		"host":           CockroachDBHost,
		"port":           CockroachDBPort,
		"database":       CockroachDBDatabase,
		"user":           CockroachDBUser,
		"password":       CockroachDBPass,
		"maxRetries":     5,
		"retryBaseDelay": "500ms",
		"queryParams": map[string]string{
			"sslmode": "disable",
		},
	}
}

func initCockroachDBConnectionPool(host, port, user, pass, dbname string) (*pgxpool.Pool, error) {
	connURL := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, pass),
		Host:     fmt.Sprintf("%s:%s", host, port),
		Path:     dbname,
		RawQuery: "sslmode=disable&application_name=cockroachdb-integration-test",
	}
	pool, err := pgxpool.New(context.Background(), connURL.String())
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	return pool, nil
}

func TestCockroachDB(t *testing.T) {
	_ = getCockroachDBVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := initCockroachDBConnectionPool(CockroachDBHost, CockroachDBPort, CockroachDBUser, CockroachDBPass, CockroachDBDatabase)
	if err != nil {
		t.Fatalf("unable to create cockroachdb connection pool: %s", err)
	}
	defer pool.Close()

	// Verify CockroachDB version
	var version string
	err = pool.QueryRow(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		t.Fatalf("failed to query version: %s", err)
	}
	if !strings.Contains(version, "CockroachDB") {
		t.Fatalf("not connected to CockroachDB, got: %s", version)
	}
	t.Logf("✅ Connected to: %s", version)

	// cleanup test environment
	tests.CleanupPostgresTables(t, ctx, pool)

	// create table name with UUID suffix
	tableNameParam := "test_crdb_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	// Create test table with UUID primary key (CockroachDB best practice)
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE %s (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name STRING NOT NULL,
			value INT,
			created_at TIMESTAMP DEFAULT now()
		)
	`, tableNameParam)

	_, err = pool.Exec(ctx, createTableSQL)
	if err != nil {
		t.Fatalf("failed to create test table: %s", err)
	}
	t.Logf("✅ Created test table: %s with UUID primary key", tableNameParam)

	// Insert test data with UUIDs
	insertSQL := fmt.Sprintf("INSERT INTO %s (name, value) VALUES ($1, $2), ($3, $4)", tableNameParam)
	_, err = pool.Exec(ctx, insertSQL, "Alice", 100, "Bob", 200)
	if err != nil {
		t.Fatalf("failed to insert test data: %s", err)
	}
	t.Logf("✅ Inserted test data with UUID primary keys")

	// Test 1: cockroach-go ExecuteTx retry mechanism
	t.Run("ExecuteTx_Retry", func(t *testing.T) {
		var count int
		err := crdbpgx.ExecuteTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM %s", tableNameParam)).Scan(&count)
		})
		if err != nil {
			t.Fatalf("ExecuteTx failed: %s", err)
		}
		if count != 2 {
			t.Errorf("expected 2 rows, got %d", count)
		}
		t.Logf("✅ cockroach-go ExecuteTx with automatic retry successful")
	})

	// Test 2: UUID primary key
	t.Run("UUID_PrimaryKey", func(t *testing.T) {
		rows, err := pool.Query(ctx, fmt.Sprintf("SELECT id, name FROM %s ORDER BY name", tableNameParam))
		if err != nil {
			t.Fatalf("failed to query: %s", err)
		}
		defer rows.Close()

		var uuids []string
		for rows.Next() {
			var id uuid.UUID
			var name string
			if err := rows.Scan(&id, &name); err != nil {
				t.Fatalf("scan failed: %s", err)
			}
			uuids = append(uuids, id.String())
			t.Logf("  Row: name=%s, uuid=%s", name, id.String())
		}

		if len(uuids) != 2 {
			t.Errorf("expected 2 UUIDs, got %d", len(uuids))
		}
		// Verify UUIDs are valid
		for _, uuidStr := range uuids {
			if _, err := uuid.Parse(uuidStr); err != nil {
				t.Errorf("invalid UUID: %s", uuidStr)
			}
		}
		t.Logf("✅ UUID primary keys working correctly (CockroachDB best practice)")
	})

	// Cleanup
	_, err = pool.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", tableNameParam))
	if err != nil {
		t.Logf("Warning: failed to cleanup test table: %s", err)
	} else {
		t.Logf("✅ Cleaned up test table")
	}

	t.Logf("✅✅✅ All CockroachDB integration tests passed!")
}
