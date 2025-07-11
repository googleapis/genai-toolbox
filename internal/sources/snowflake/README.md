# Snowflake Database Support

This package provides support for connecting to Snowflake databases in the genai-toolbox.

## Configuration

### Source Configuration

Configure a Snowflake source in your tools configuration file:

```yaml
sources:
  my-snowflake-db:
    kind: snowflake
    account: your-account
    user: your-username
    password: your-password
    database: your-database
    schema: your-schema
```

### Required Parameters

- `account`: Your Snowflake account identifier
- `user`: Username for authentication
- `password`: Password for authentication
- `database`: Database name to connect to
- `schema`: Schema name within the database

### Environment Variables

You can use environment variables in your configuration:

```yaml
sources:
  my-snowflake-db:
    kind: snowflake
    account: ${SNOWFLAKE_ACCOUNT}
    user: ${SNOWFLAKE_USER}
    password: ${SNOWFLAKE_PASSWORD}
    database: ${SNOWFLAKE_DATABASE}
    schema: ${SNOWFLAKE_SCHEMA}
```

## Tools

### snowflake-execute-sql

Execute arbitrary SQL statements against your Snowflake database:

```yaml
tools:
  execute_sql:
    kind: snowflake-execute-sql
    source: my-snowflake-db
    description: Execute SQL statements on Snowflake
```

### snowflake-sql

Execute parameterized SQL statements with predefined queries:

```yaml
tools:
  get_user_by_id:
    kind: snowflake-sql
    source: my-snowflake-db
    description: Get user information by ID
    statement: SELECT * FROM users WHERE id = $1
    parameters:
      - name: user_id
        type: string
        description: The user ID to look up
```

## Usage Example

### Complete Configuration

```yaml
sources:
  snowflake-db:
    kind: snowflake
    account: ${SNOWFLAKE_ACCOUNT}
    user: ${SNOWFLAKE_USER}
    password: ${SNOWFLAKE_PASSWORD}
    database: ${SNOWFLAKE_DATABASE}
    schema: ${SNOWFLAKE_SCHEMA}

tools:
  execute_sql:
    kind: snowflake-execute-sql
    source: snowflake-db
    description: Execute SQL on Snowflake

  list_tables:
    kind: snowflake-sql
    source: snowflake-db
    description: List all tables in the current schema
    statement: |
      SELECT table_name, table_type 
      FROM information_schema.tables 
      WHERE table_schema = current_schema()
      ORDER BY table_name
```

### Using the Prebuilt Configuration

You can also use the prebuilt configuration:

```bash
export SNOWFLAKE_ACCOUNT=your-account
export SNOWFLAKE_USER=your-username
export SNOWFLAKE_PASSWORD=your-password
export SNOWFLAKE_DATABASE=your-database
export SNOWFLAKE_SCHEMA=your-schema

toolbox --prebuilt snowflake
```

## Connection Details

The Snowflake source uses the following connection parameters:

- **Driver**: `github.com/snowflakedb/gosnowflake`
- **Connection pooling**: Managed by `github.com/jmoiron/sqlx`
- **Default warehouse**: `COMPUTE_WH`
- **Default role**: `ACCOUNTADMIN`
- **Protocol**: HTTPS
- **Timeout**: 60 seconds

## Security Notes

- Store sensitive credentials in environment variables
- Use appropriate Snowflake roles with minimal required privileges
- Consider using key-pair authentication for production environments
- Ensure your Snowflake account has proper network access policies configured
