package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMCPPostgresListViewsTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	viewName := "test_view_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	dropViewfunc1 := setUpPostgresViews(t, ctx, pool, viewName)
	defer dropViewfunc1()

	invokeTcs := []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "invoke list_views with newly created view",
			args: map[string]any{"view_name": viewName},
			want: fmt.Sprintf(`[{"schema_name":"public","view_name":"%s","owner_name":"postgres","definition":" SELECT 1 AS col;"}]`, viewName),
		},
		{
			name: "invoke list_views with non-existent_view",
			args: map[string]any{"view_name": "non_existent_view"},
			want: `null`,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			RunMCPCustomToolCallMethod(t, "list_views", tc.args, tc.want)
		})
	}
}

func RunMCPPostgresListSchemasTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool, owner string, uniqueID string) {
	schemaName := "test_schema_" + uniqueID
	cleanup := setupPostgresSchemas(t, ctx, pool, schemaName)
	defer cleanup()

	wantSchema := fmt.Sprintf(`[{"functions":0,"grants":{},"owner":"%s","schema_name":"%s","tables":0,"views":0}]`, owner, schemaName)

	invokeTcs := []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "invoke list_schemas with schema_name",
			args: map[string]any{"schema_name": schemaName},
			want: wantSchema,
		},
		{
			name: "invoke list_schemas with limit 1",
			args: map[string]any{"schema_name": schemaName, "limit": 1},
			want: wantSchema,
		},
		{
			name: "invoke list_schemas with non-existent schema",
			args: map[string]any{"schema_name": "non_existent_schema"},
			want: `null`,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			RunMCPCustomToolCallMethod(t, "list_schemas", tc.args, tc.want)
		})
	}
}

func RunMCPPostgresListActiveQueriesTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	invokeTcs := []struct {
		name                string
		args                map[string]any
		clientSleepSecs     int
		waitSecsBeforeCheck int
		want                string
	}{
		{
			name:                "invoke list_active_queries when the system is idle",
			args:                map[string]any{"exclude_application_names": "wal_uploader"},
			clientSleepSecs:     0,
			waitSecsBeforeCheck: 0,
			want:                `"null"`,
		},
		{
			name:                "invoke list_active_queries when there is 1 ongoing but lower than the threshold",
			args:                map[string]any{"min_duration": "100 seconds", "exclude_application_names": "wal_uploader"},
			clientSleepSecs:     1,
			waitSecsBeforeCheck: 1,
			want:                `"null"`,
		},
		{
			name:                "invoke list_active_queries when 1 ongoing query should show up",
			args:                map[string]any{"min_duration": "1 seconds", "exclude_application_names": "wal_uploader"},
			clientSleepSecs:     10,
			waitSecsBeforeCheck: 5,
			want:                `"SELECT pg_sleep(10);"`,
		},
	}

	var wg sync.WaitGroup
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			if tc.clientSleepSecs > 0 {
				wg.Add(1)

				go func() {
					defer wg.Done()

					err := pool.Ping(ctx)
					if err != nil {
						t.Errorf("unable to connect to test database: %s", err)
						return
					}
					_, err = pool.Exec(ctx, fmt.Sprintf("SELECT pg_sleep(%d);", tc.clientSleepSecs))
					if err != nil {
						t.Errorf("Executing 'SELECT pg_sleep' failed: %s", err)
					}
				}()
			}

			if tc.waitSecsBeforeCheck > 0 {
				time.Sleep(time.Duration(tc.waitSecsBeforeCheck) * time.Second)
			}

			RunMCPCustomToolCallMethod(t, "list_active_queries", tc.args, tc.want)
		})
	}
	wg.Wait()
}

func RunMCPPostgresListAvailableExtensionsTest(t *testing.T) {
	t.Run("invoke list_available_extensions output", func(t *testing.T) {
		RunMCPCustomToolCallMethod(t, "list_available_extensions", map[string]any{}, "")
	})
}

func RunMCPPostgresListInstalledExtensionsTest(t *testing.T) {
	t.Run("invoke list_installed_extensions output", func(t *testing.T) {
		RunMCPCustomToolCallMethod(t, "list_installed_extensions", map[string]any{}, "")
	})
}

func RunMCPPostgresDatabaseOverviewTest(t *testing.T) {
	t.Run("invoke database_overview output", func(t *testing.T) {
		code, mcpResp, err := InvokeMCPTool(t, "database_overview", map[string]any{}, nil)
		if err != nil {
			t.Fatalf("failed to invoke tool: %v", err)
		}
		if code != 200 {
			t.Fatalf("unexpected status code: got %d, want 200", code)
		}
		if len(mcpResp.Result.Content) == 0 {
			t.Fatalf("empty content in response")
		}
		text := mcpResp.Result.Content[0].Text

		expectedKeys := []string{
			`"pg_version"`,
			`"is_replica"`,
			`"uptime"`,
			`"max_connections"`,
			`"current_connections"`,
			`"active_connections"`,
			`"pct_connections_used"`,
		}

		for _, key := range expectedKeys {
			if !strings.Contains(text, key) {
				t.Errorf("Missing expected key in result: %s", key)
			}
		}
	})
}

func RunMCPPostgresListTriggersTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	uniqueID := strings.ReplaceAll(uuid.New().String(), "-", "")
	schemaName := "test_schema_" + uniqueID
	tableName := "test_table_" + uniqueID
	functionName := "test_func_" + uniqueID
	triggerName := "test_trigger_" + uniqueID

	cleanup := setupPostgresTrigger(t, ctx, pool, schemaName, tableName, functionName, triggerName)
	defer cleanup()

	// Definition can vary slightly based on server version/settings, so we fetch it to compare.
	var expectedDef string
	getDefQuery := fmt.Sprintf("SELECT pg_get_triggerdef(oid) FROM pg_trigger WHERE tgname = '%s'", triggerName)
	err := pool.QueryRow(ctx, getDefQuery).Scan(&expectedDef)
	if err != nil {
		t.Fatalf("failed to fetch trigger definition: %v", err)
	}

	wantTrigger := fmt.Sprintf(`"trigger_name":"%s"`, triggerName)

	invokeTcs := []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "list all triggers (expecting the one we created)",
			args: map[string]any{},
			want: wantTrigger,
		},
		{
			name: "filter by trigger_name",
			args: map[string]any{"trigger_name": triggerName},
			want: wantTrigger,
		},
		{
			name: "filter by schema_name",
			args: map[string]any{"schema_name": schemaName},
			want: wantTrigger,
		},
		{
			name: "filter by table_name",
			args: map[string]any{"table_name": tableName},
			want: wantTrigger,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			RunMCPCustomToolCallMethod(t, "list_triggers", tc.args, tc.want)
		})
	}
}

func RunMCPPostgresListIndexesTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	schemaName := "testschema_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableName := "table1_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	cleanup := setupPostgresIndex(t, ctx, pool, schemaName, tableName)
	defer cleanup(t)

	wantIndexStr := tableName + "_email_idx"

	invokeTcs := []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "list_indexes for a specific schema and table",
			args: map[string]any{"schema_name": schemaName, "table_name": tableName},
			want: wantIndexStr,
		},
		{
			name: "list_indexes for a specific schema",
			args: map[string]any{"schema_name": schemaName},
			want: wantIndexStr,
		},
		{
			name: "list_indexes with non-existent schema",
			args: map[string]any{"schema_name": "non_existent_schema"},
			want: `"null"`,
		},
		{
			name: "list_indexes with non-existent table in existing schema",
			args: map[string]any{"schema_name": schemaName, "table_name": "non_existent_table"},
			want: `"null"`,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			RunMCPCustomToolCallMethod(t, "list_indexes", tc.args, tc.want)
		})
	}
}

func RunMCPPostgresListSequencesTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	sequenceName, teardown := setupListSequencesTest(t, ctx, pool)
	defer teardown(t)

	invokeTcs := []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "invoke list_sequences",
			args: map[string]any{"sequence_name": sequenceName},
			want: sequenceName,
		},
		{
			name: "invoke list_sequences with non-existent sequence",
			args: map[string]any{"sequence_name": "non_existent_sequence"},
			want: `"null"`,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			RunMCPCustomToolCallMethod(t, "list_sequences", tc.args, tc.want)
		})
	}
}

func RunMCPPostgresLongRunningTransactionsTest(t *testing.T) {
	invokeTcs := []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "invoke long_running_transactions with default threshold",
			args: map[string]any{},
			want: "",
		},
		{
			name: "invoke long_running_transactions with custom threshold",
			args: map[string]any{"min_transaction_duration_secs": 3600},
			want: "",
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			RunMCPCustomToolCallMethod(t, "long_running_transactions", tc.args, tc.want)
		})
	}
}

func RunMCPPostgresListLocksTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	tableName := "test_postgres_list_locks_table"
	cleanup := CreateAndLockPostgresTable(t, ctx, pool, tableName)
	defer cleanup()

	t.Run("invoke list_locks with no arguments", func(t *testing.T) {
		RunMCPCustomToolCallMethod(t, "list_locks", map[string]any{}, tableName)
	})
}

func RunMCPPostgresReplicationStatsTest(t *testing.T) {
	t.Run("invoke replication_stats with no arguments", func(t *testing.T) {
		RunMCPCustomToolCallMethod(t, "replication_stats", map[string]any{}, "")
	})
}

func RunMCPPostgresGetColumnCardinalityTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	schemaName := "testschema_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableName := "table1_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	cleanup := setupPostgresSchemas(t, ctx, pool, schemaName)
	defer cleanup()

	// Create table with multiple columns
	createTableStmt := fmt.Sprintf(`
		CREATE TABLE %s.%s (
			id SERIAL PRIMARY KEY,
			email VARCHAR(100) UNIQUE,
			name VARCHAR(50),
			status VARCHAR(20),
			created_at TIMESTAMP
		)
	`, schemaName, tableName)

	if _, err := pool.Exec(ctx, createTableStmt); err != nil {
		t.Fatalf("unable to create table: %s", err)
	}

	// Insert larger sample data to ensure statistics are collected
	insertStmt := fmt.Sprintf(`
		INSERT INTO %s.%s (email, name, status, created_at) VALUES
		('user1@example.com', 'Alice', 'active', NOW()),
		('user2@example.com', 'Bob', 'inactive', NOW()),
		('user3@example.com', 'Charlie', 'active', NOW())
	`, schemaName, tableName)

	if _, err := pool.Exec(ctx, insertStmt); err != nil {
		t.Fatalf("unable to insert data: %s", err)
	}

	// Run ANALYZE to update statistics
	analyzeStmt := fmt.Sprintf(`ANALYZE %s.%s`, schemaName, tableName)
	if _, err := pool.Exec(ctx, analyzeStmt); err != nil {
		t.Fatalf("unable to run ANALYZE: %s", err)
	}

	invokeTcs := []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "get cardinality for a specific column",
			args: map[string]any{"schema_name": schemaName, "table_name": tableName, "column_name": "email"},
			want: `"column_name": "email"`,
		},
		{
			name: "get cardinality for all columns",
			args: map[string]any{"schema_name": schemaName, "table_name": tableName},
			want: `"column_name"`,
		},
		{
			name: "get cardinality with non-existent column",
			args: map[string]any{"schema_name": schemaName, "table_name": tableName, "column_name": "non_existent"},
			want: ``,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			RunMCPCustomToolCallMethod(t, "get_column_cardinality", tc.args, tc.want)
		})
	}
}

func RunMCPPostgresListTableStatsTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	testTableName := "test_list_table_stats_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	createTableStmt := fmt.Sprintf(`
        CREATE TABLE %s (
            id SERIAL PRIMARY KEY,
            name VARCHAR(100),
            email VARCHAR(100)
        )
    `, testTableName)

	if _, err := pool.Exec(ctx, createTableStmt); err != nil {
		t.Fatalf("unable to create test table: %s", err)
	}
	defer func() {
		dropTableStmt := fmt.Sprintf("DROP TABLE IF EXISTS %s", testTableName)
		if _, err := pool.Exec(ctx, dropTableStmt); err != nil {
			t.Logf("warning: unable to drop test table: %v", err)
		}
	}()

	// Insert some data to generate statistics
	insertStmt := fmt.Sprintf(`
        INSERT INTO %s (name, email) VALUES
        ('Alice', 'alice@example.com'),
        ('Bob', 'bob@example.com')
    `, testTableName)

	if _, err := pool.Exec(ctx, insertStmt); err != nil {
		t.Fatalf("unable to insert test data: %s", err)
	}

	// Run ANALYZE to update statistics
	analyzeStmt := fmt.Sprintf("ANALYZE %s", testTableName)
	if _, err := pool.Exec(ctx, analyzeStmt); err != nil {
		t.Logf("warning: unable to run ANALYZE: %v", err)
	}

	invokeTcs := []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "list table stats with no arguments (default limit)",
			args: map[string]any{},
			want: "",
		},
		{
			name: "list table stats filtering by specific table",
			args: map[string]any{"table_name": testTableName},
			want: testTableName,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			RunMCPCustomToolCallMethod(t, "list_table_stats", tc.args, tc.want)
		})
	}
}

func RunMCPPostgresListPublicationTablesTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	table1Name := "pub_table_1"
	pub1Name := "pub_1"

	table2Name := "pub_table_2"
	pub2Name := "pub_2"

	cleanup := setupPostgresPublicationTable(t, ctx, pool, table1Name, pub1Name)
	defer cleanup(t)
	cleanup2 := setupPostgresPublicationTable(t, ctx, pool, table2Name, pub2Name)
	defer cleanup2(t)

	invokeTcs := []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "list all publication tables",
			args: map[string]any{},
			want: pub1Name,
		},
		{
			name: "list all tables for the created publication",
			args: map[string]any{"publication_names": pub1Name},
			want: pub1Name,
		},
		{
			name: "filter by table_name",
			args: map[string]any{"table_names": table1Name},
			want: pub1Name,
		},
		{
			name: "invoke list_publication_tables with non-existent table",
			args: map[string]any{"table_names": "non_existent_table"},
			want: ``,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			RunMCPCustomToolCallMethod(t, "list_publication_tables", tc.args, tc.want)
		})
	}
}

func RunMCPPostgresListTableSpacesTest(t *testing.T) {
	t.Run("invoke list_tablespaces output", func(t *testing.T) {
		RunMCPCustomToolCallMethod(t, "list_tablespaces", map[string]any{}, "")
	})
}

func RunMCPPostgresListPgSettingsTest(t *testing.T) {
	targetSetting := "maintenance_work_mem"
	invokeTcs := []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "invoke list_pg_settings with specific setting",
			args: map[string]any{"setting_name": targetSetting},
			want: targetSetting,
		},
		{
			name: "invoke list_pg_settings with non-existent setting",
			args: map[string]any{"setting_name": "non_existent_config_xyz"},
			want: `"null"`,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			RunMCPCustomToolCallMethod(t, "list_pg_settings", tc.args, tc.want)
		})
	}
}

func RunMCPPostgresListDatabaseStatsTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	dbName1 := "test_db_stats_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	dbOwner1 := "test_user_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	dbName2 := "test_db_stats_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	dbOwner2 := "test_user_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	cleanup1 := setUpDatabase(t, ctx, pool, dbName1, dbOwner1)
	defer cleanup1()
	cleanup2 := setUpDatabase(t, ctx, pool, dbName2, dbOwner2)
	defer cleanup2()

	invokeTcs := []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "invoke database_stats filtering by specific database name",
			args: map[string]any{"database_name": dbName1},
			want: dbName1,
		},
		{
			name: "invoke database_stats filtering by specific owner",
			args: map[string]any{"database_owner": dbOwner2},
			want: dbName2,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			RunMCPCustomToolCallMethod(t, "list_database_stats", tc.args, tc.want)
		})
	}
}

func RunMCPPostgresListRolesTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	adminUser, superUser, normalUser, cleanup := setupPostgresRoles(t, ctx, pool)
	_ = normalUser
	defer cleanup(t)

	invokeTcs := []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "list_roles with filter for created roles",
			args: map[string]any{"role_name": "test_role_"},
			want: adminUser,
		},
		{
			name: "list_roles filter specific role",
			args: map[string]any{"role_name": superUser},
			want: superUser,
		},
		{
			name: "list_roles non-existent role",
			args: map[string]any{"role_name": "non_existent_role_xyz"},
			want: `"null"`,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			RunMCPCustomToolCallMethod(t, "list_roles", tc.args, tc.want)
		})
	}
}

func RunMCPPostgresListStoredProcedureTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	// Create test schema
	now := time.Now().Unix()
	testSchemaName := fmt.Sprintf("test_proc_%d_%s", now, strings.ReplaceAll(uuid.New().String(), "-", "")[:8])
	createSchemaStmt := fmt.Sprintf("CREATE SCHEMA %s", testSchemaName)
	if _, err := pool.Exec(ctx, createSchemaStmt); err != nil {
		t.Fatalf("unable to create test schema: %v", err)
	}
	defer func() {
		dropSchemaStmt := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", testSchemaName)
		if _, err := pool.Exec(ctx, dropSchemaStmt); err != nil {
			t.Logf("warning: unable to drop test schema: %v", err)
		}
	}()

	// Create test procedures
	proc1Name := "test_proc_1_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	createProc1Stmt := fmt.Sprintf(`
        CREATE PROCEDURE %s.%s(p_count INT)
        LANGUAGE plpgsql
        AS $$
        BEGIN
            COMMIT;
        END;
        $$
    `, testSchemaName, proc1Name)

	if _, err := pool.Exec(ctx, createProc1Stmt); err != nil {
		t.Fatalf("unable to create test procedure 1: %v", err)
	}

	invokeTcs := []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "list stored procedures filtering by specific schema",
			args: map[string]any{"schema_name": testSchemaName},
			want: proc1Name,
		},
		{
			name: "list stored procedures with non-existent schema",
			args: map[string]any{"schema_name": "non_existent_schema_xyz"},
			want: ``,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			RunMCPCustomToolCallMethod(t, "list_stored_procedure", tc.args, tc.want)
		})
	}
}

func RunMCPPostgresListTablesTest(t *testing.T, tableNameParam, tableNameAuth, user string) {
	// TableNameParam columns to construct want
	paramTableColumns := fmt.Sprintf(`[
		{"data_type": "integer", "column_name": "id", "column_default": "nextval('%[1]s_id_seq'::regclass)", "is_not_nullable": true, "ordinal_position": 1, "column_comment": null},
		{"data_type": "text", "column_name": "name", "column_default": null, "is_not_nullable": false, "ordinal_position": 2, "column_comment": null}
	]`, tableNameParam)

	// TableNameAuth columns to construct want
	authTableColumns := fmt.Sprintf(`[
		{"data_type": "integer", "column_name": "id", "column_default": "nextval('%[1]s_id_seq'::regclass)", "is_not_nullable": true, "ordinal_position": 1, "column_comment": null},
		{"data_type": "text", "column_name": "name", "column_default": null, "is_not_nullable": false, "ordinal_position": 2, "column_comment": null},
		{"data_type": "text", "column_name": "email", "column_default": null, "is_not_nullable": false, "ordinal_position": 3, "column_comment": null}
	]`, tableNameAuth)

	const (
		// Template to construct detailed output want
		detailedObjectTemplate = `{
            "object_name": "%[1]s", "schema_name": "public",
            "object_details": {
                "owner": "%[3]s", "comment": null,
                "indexes": [{"is_primary": true, "is_unique": true, "index_name": "%[1]s_pkey", "index_method": "btree", "index_columns": ["id"], "index_definition": "CREATE UNIQUE INDEX %[1]s_pkey ON public.%[1]s USING btree (id)"}],
                "triggers": [], "columns": %[2]s, "object_name": "%[1]s", "object_type": "TABLE", "schema_name": "public",
                "constraints": [{"constraint_name": "%[1]s_pkey", "constraint_type": "PRIMARY KEY", "constraint_columns": ["id"], "constraint_definition": "PRIMARY KEY (id)", "foreign_key_referenced_table": null, "foreign_key_referenced_columns": null}]
            }
        }`

		// Template to construct simple output want
		simpleObjectTemplate = `{"object_name":"%s", "schema_name":"public", "object_details":{"name":"%s"}}`
	)

	// Helper to build json for detailed want
	getDetailedWant := func(tableName, columnJSON string) string {
		return fmt.Sprintf(detailedObjectTemplate, tableName, columnJSON, user)
	}

	// Helper to build template for simple want
	getSimpleWant := func(tableName string) string {
		return fmt.Sprintf(simpleObjectTemplate, tableName, tableName)
	}

	invokeTcs := []struct {
		name string
		args map[string]any
		want []string
	}{
		{
			name: "invoke list_tables all tables detailed output",
			args: map[string]any{"table_names": ""},
			want: []string{getDetailedWant(tableNameAuth, authTableColumns), getDetailedWant(tableNameParam, paramTableColumns)},
		},
		{
			name: "invoke list_tables all tables simple output",
			args: map[string]any{"table_names": "", "output_format": "simple"},
			want: []string{getSimpleWant(tableNameAuth), getSimpleWant(tableNameParam)},
		},
		{
			name: "invoke list_tables detailed output",
			args: map[string]any{"table_names": tableNameAuth},
			want: []string{getDetailedWant(tableNameAuth, authTableColumns)},
		},
		{
			name: "invoke list_tables simple output",
			args: map[string]any{"table_names": tableNameAuth, "output_format": "simple"},
			want: []string{getSimpleWant(tableNameAuth)},
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, mcpResp, err := InvokeMCPTool(t, "list_tables", tc.args, nil)
			if err != nil {
				t.Fatalf("native error executing list_tables: %s", err)
			}
			if statusCode != http.StatusOK {
				t.Fatalf("expected status 200, got %d", statusCode)
			}
			if mcpResp.Result.IsError {
				t.Fatalf("list_tables returned error result: %v", mcpResp.Result)
			}
			if len(mcpResp.Result.Content) == 0 {
				t.Fatalf("list_tables returned empty content field")
			}
			got := mcpResp.Result.Content[0].Text

			for _, wantStr := range tc.want {
				contains, err := semanticJSONContains(got, wantStr)
				if err != nil {
					t.Fatalf("error in semantic JSON check: %v", err)
				}
				if !contains {
					t.Fatalf("expected %q to contain %q (semantic match)", got, wantStr)
				}
			}
		})
	}
}

// semanticJSONContains checks if wantJSON is semantically equal to gotJSON or contained in it if gotJSON is a list.
func semanticJSONContains(gotJSON, wantJSON string) (bool, error) {
	var gotList []any
	if err := json.Unmarshal([]byte(gotJSON), &gotList); err != nil {
		// If not a list, try as a single object
		var gotObj any
		if err := json.Unmarshal([]byte(gotJSON), &gotObj); err != nil {
			return false, fmt.Errorf("error unmarshaling got: %w", err)
		}
		var wantObj any
		if err := json.Unmarshal([]byte(wantJSON), &wantObj); err != nil {
			return false, fmt.Errorf("error unmarshaling want: %w", err)
		}
		return reflect.DeepEqual(gotObj, wantObj), nil
	}

	// It is a list!
	var wantObj any
	if err := json.Unmarshal([]byte(wantJSON), &wantObj); err != nil {
		return false, fmt.Errorf("error unmarshaling want: %w", err)
	}

	for _, gotObj := range gotList {
		if reflect.DeepEqual(gotObj, wantObj) {
			return true, nil
		}
	}
	return false, nil
}

func RunMCPPostgresTemplateToolsTest(t *testing.T, tableName string) {
	t.Run("MCP template tools - create table", func(t *testing.T) {
		args := map[string]any{
			"tableName": tableName,
			"columns":   []string{"id serial PRIMARY KEY", "name VARCHAR(50)", "age INT"},
		}
		RunMCPCustomToolCallMethod(t, "create-table-templateParams-tool", args, "null")
	})

	t.Run("MCP template tools - insert table", func(t *testing.T) {
		args := map[string]any{
			"tableName": tableName,
			"columns":   []string{"id", "name", "age"},
			"values":    "1, 'Alex', 21",
		}
		RunMCPCustomToolCallMethod(t, "insert-table-templateParams-tool", args, "null")
	})
}

func RunMCPPostgresListQueryStatsTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	// Insert a simple query by running a SELECT statement
	// This will record statistics in pg_stat_statements
	selectStmt := "SELECT 1 as test_query"
	if _, err := pool.Exec(ctx, selectStmt); err != nil {
		t.Logf("warning: unable to execute test query: %s", err)
	}

	dropExtensionFunc := createPostgresExtension(t, ctx, pool, "pg_stat_statements")
	defer dropExtensionFunc()

	t.Run("MCP list query stats - default limit", func(t *testing.T) {
		RunMCPCustomToolCallMethod(t, "list_query_stats", map[string]any{}, `"test_query"`)
	})

	t.Run("MCP list query stats - custom limit", func(t *testing.T) {
		RunMCPCustomToolCallMethod(t, "list_query_stats", map[string]any{"limit": 10}, `"test_query"`)
	})

	t.Run("MCP list query stats - specific database", func(t *testing.T) {
		RunMCPCustomToolCallMethod(t, "list_query_stats", map[string]any{"database_name": "postgres"}, `"test_query"`)
	})

	t.Run("MCP list query stats - non-existent database", func(t *testing.T) {
		RunMCPCustomToolCallMethod(t, "list_query_stats", map[string]any{"database_name": "non_existent_db_xyz"}, `null`)
	})
}
