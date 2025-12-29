# ClickHouse MCP Server

The ClickHouse Model Context Protocol (MCP) Server enables AI-powered development tools to seamlessly connect, interact, and generate data insights with your ClickHouse databases using natural language commands.

## Features

An editor configured to use the ClickHouse MCP server can use its AI capabilities to help you:

- **Natural Language to Data Analytics:** Easily find required ClickHouse tables and ask analytical questions in plain English.
- **Seamless Workflow:** Stay within your CLI, eliminating the need to constantly switch to the ClickHouse console for generating analytical insights.
- **Execute SQL Queries:** Run parameterized SQL queries and prepared statements against your ClickHouse databases.


## Server Capabilities

The ClickHouse MCP server provides the following tools:

| Tool Name              | Description                                                     |
|:-----------------------|:----------------------------------------------------------------|
| `execute_sql`          | Executes a SQL query against ClickHouse.                      |
| `list_databases`       | Lists all databases in the ClickHouse instance.                |
| `list_tables`          | Lists all tables in a specific ClickHouse database.            |

## Custom MCP Server Configuration

The ClickHouse MCP server is configured using environment variables.

```bash
export CLICKHOUSE_HOST="<your-clickhouse-host>"
export CLICKHOUSE_PORT="<your-clickhouse-port>"  # e.g., "8123" for HTTP, "9000" for native
export CLICKHOUSE_USER="<your-clickhouse-user>"
export CLICKHOUSE_PASSWORD="<your-clickhouse-password>"
export CLICKHOUSE_DATABASE="<your-database-name>"
export CLICKHOUSE_PROTOCOL="https"  # Optional: "http" or "https" (default: "https")
```

Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI, `mcp_config.json` for Antigravity):

```json
{
  "mcpServers": {
    "clickhouse": {
      "command": "npx",
      "args": ["-y", "@toolbox-sdk/server", "--prebuilt", "clickhouse", "--stdio"],
      "env": {
        "CLICKHOUSE_HOST": "<your-clickhouse-host>",
        "CLICKHOUSE_PORT": "<your-clickhouse-port>",
        "CLICKHOUSE_USER": "<your-clickhouse-user>",
        "CLICKHOUSE_PASSWORD": "<your-clickhouse-password>",
        "CLICKHOUSE_DATABASE": "<your-database-name>",
        "CLICKHOUSE_PROTOCOL": "https"
      }
    }
  }
}
```

### Advanced Configuration

You can also configure connection pool settings in your `tools.yaml` file:

```yaml
sources:
  my-clickhouse-source:
    kind: clickhouse
    host: ${CLICKHOUSE_HOST}
    port: ${CLICKHOUSE_PORT}
    database: ${CLICKHOUSE_DATABASE}
    user: ${CLICKHOUSE_USER}
    password: ${CLICKHOUSE_PASSWORD}
    protocol: https  # Optional: http or https (default: https)
    secure: true     # Optional: boolean (default: false)
    # Optional connection pool settings
    maxOpenConns: 50        # Optional: Maximum number of open connections (default: 25)
    maxIdleConns: 10       # Optional: Maximum number of idle connections (default: 5)
    connMaxLifetime: 10m   # Optional: Maximum connection lifetime (default: 5m)
                            # Accepts duration strings like "30s", "5m", "1h", etc.
```
