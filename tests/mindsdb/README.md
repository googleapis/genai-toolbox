# MindsDB Integration Test

This directory contains a focused MindsDB integration test that demonstrates testing the `mindsdb-execute-sql` tool using MindsDB's built-in `files` schema.

## Test Overview

The `mindsdb_integration_test.go` file contains two comprehensive tests:

### TestMindsDBToolEndpoints
Tests the generic tool functionality including:
- Tool GET endpoints
- Basic tool invocation
- Parameterized tool invocation (mindsdb-sql tool)
- Execute SQL tool functionality using standard `my-exec-sql-tool` pattern
- Auth features and edge cases

### TestMindsDBExecuteSQLTool  
Tests the `mindsdb-execute-sql` tool with the following SQL operations:

1. **CREATE TABLE** - Creates a test table in the files schema
2. **INSERT** - Adds test data to the table
3. **SELECT** - Queries data from the table
4. **UPDATE** - Modifies existing records
5. **DELETE** - Removes records from the table
6. **DROP TABLE** - Cleans up by dropping the test table

## Key Features

### Standard Tool Pattern
- Uses the standard `my-exec-sql-tool` pattern consistent with other integration tests (BigQuery, Spanner, etc.)
- Follows the same configuration and testing patterns as other database integrations
- Ensures consistency across the test suite

### Files Schema Usage
- Uses MindsDB's built-in `files` schema instead of external databases
- No MySQL dependencies or complex setup required
- Simple and fast test execution

### Optional Authentication
- Password is optional (only added if `MINDSDB_PASS` environment variable is set)
- Works with both password-protected and password-free MindsDB instances

### Comprehensive SQL Testing
```go
// Test CREATE TABLE
createTableStatement := fmt.Sprintf(`"CREATE TABLE files.%s (id INT PRIMARY KEY, name VARCHAR(255), email VARCHAR(255))"`, tableName)

// Test INSERT
insertStatement := fmt.Sprintf(`"INSERT INTO files.%s (id, name, email) VALUES (1, 'Alice', 'alice@example.com'), (2, 'Bob', 'bob@example.com')"`, tableName)

// Test SELECT
tests.RunExecuteSqlToolInvokeTest(t, fmt.Sprintf(`"SELECT * FROM files.%s;"`, tableName), selectTableWant)

// Test UPDATE
updateStatement := fmt.Sprintf(`"UPDATE files.%s SET name = 'Alice Updated' WHERE id = 1"`, tableName)

// Test DELETE
deleteStatement := fmt.Sprintf(`"DELETE FROM files.%s WHERE id = 2"`, tableName)

// Test DROP TABLE
dropTableStatement := fmt.Sprintf(`"DROP TABLE files.%s"`, tableName)
```

## Environment Variables

Required environment variables:
```bash
export MINDSDB_DATABASE="files"
export MINDSDB_HOST="127.0.0.1"
export MINDSDB_PORT="47335"
export MINDSDB_USER="mindsdb"
```

Optional environment variable:
```bash
export MINDSDB_PASS="mindsdb"  # Only if authentication is required
```

## Example Configuration

Here's a complete YAML configuration example for the MindsDB execute-sql tool:

```yaml
sources:
  my-mindsdb-source:
    kind: mindsdb
    host: 127.0.0.1
    port: 47335
    database: files
    user: mindsdb
    # password: mindsdb  # Optional - only if authentication is required

tools:
  my-exec-sql-tool:
    kind: mindsdb-execute-sql
    source: my-mindsdb-source
    description: |
      Execute SQL queries directly on MindsDB database.
      Use this tool to run any SQL statement against your MindsDB instance.
      Example: SELECT * FROM my_table LIMIT 10
```

## Test Configuration

The test uses the standard configuration pattern with `my-exec-sql-tool`:

```go
"tools": map[string]any{
    "my-exec-sql-tool": map[string]any{
        "kind":        "mindsdb-execute-sql",
        "source":      "my-instance",
        "description": "Tool to execute sql",
    },
    "my-auth-exec-sql-tool": map[string]any{
        "kind":        "mindsdb-execute-sql",
        "source":      "my-instance",
        "description": "Tool to execute sql",
        "authRequired": []string{
            "my-google-auth",
        },
    },
}
```

### Tool Parameters

The `mindsdb-execute-sql` tool takes a single parameter:

- **`sql`** (string): The SQL query to execute

Example usage:
```json
{
  "sql": "SELECT * FROM files.my_table LIMIT 10"
}
```

### Supported SQL Operations

The MindsDB files schema supports the following SQL operations:

- **SELECT** - Query data from tables
- **INSERT** - Add new records to tables  
- **CREATE TABLE** - Create new tables
- **DROP TABLE** - Remove tables

**Note**: UPDATE and DELETE operations are not supported in the files schema.

## Benefits

- **Standard Pattern**: Follows the same `my-exec-sql-tool` pattern as other integration tests
- **Simple Setup**: No external database dependencies
- **Fast Execution**: Direct connection to MindsDB
- **Comprehensive**: Tests all major SQL operations
- **Flexible**: Works with or without authentication
- **Clean**: Proper table creation and cleanup
- **Focused**: Only tests the execute-sql functionality
- **Consistent**: Matches the testing patterns used by BigQuery, Spanner, and other integrations

## Running the Test

```bash
# Set required environment variables
export MINDSDB_DATABASE="files"
export MINDSDB_HOST="127.0.0.1"
export MINDSDB_PORT="47335"
export MINDSDB_USER="mindsdb"

# Run the test
go test ./tests/mindsdb -v
```

The test will create temporary tables, perform various SQL operations on them, and clean up afterward, providing comprehensive validation of both the `mindsdb-execute-sql` tool and the generic test functionality including auth features and edge cases, using the standard `my-exec-sql-tool` pattern consistent with other integration tests. 