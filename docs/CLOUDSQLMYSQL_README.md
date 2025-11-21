# Cloud SQL for MySQL MCP Server

The Cloud SQL for MySQL Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud SQL for MySQL databases. It supports connecting to instances, exploring schemas, and running queries.

## Features

An editor configured to use the Cloud SQL for MySQL MCP server can use its AI capabilities to help you:

- **Query Data** - Execute SQL queries and analyze query plans
- **Explore Schema** - List tables and view schema details
- **Database Maintenance** - Check for fragmentation and missing indexes
- **Monitor Performance** - View active queries

## Installation and Setup

### Prerequisites

*   Download and install [MCP Toolbox](https://github.com/googleapis/genai-toolbox):
  1.  **Download the Toolbox binary**:
      Download the latest binary for your operating system and architecture from the storage bucket. Check the [releases page](https://github.com/googleapis/genai-toolbox/releases) for OS and CPU architecture support:
      `https://storage.googleapis.com/genai-toolbox/v0.20.0/<os>/<arch>/toolbox`
      *   Replace `<os>` with `linux`, `darwin` (macOS), or `windows`.
      *   Replace `<arch>` with `amd64` (Intel) or `arm64` (Apple Silicon).
      
      

  2.  **Make it executable**:
      ```bash
      chmod +x toolbox
      ```

  3.  **Add the binary to $PATH in `.~/bash_profile`**:
      ```bash
      export PATH=$PATH:/path/to/toolbox
      ```
    
You may need to restart Antigravity for changes to take effect.

*   A Google Cloud project with the **Cloud SQL Admin API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   Cloud SQL Client (`roles/cloudsql.client`)

### Configuration

The MCP server is configured using environment variables.

```bash
export CLOUD_SQL_MYSQL_PROJECT="<your-gcp-project-id>"
export CLOUD_SQL_MYSQL_REGION="<your-cloud-sql-region>"
export CLOUD_SQL_MYSQL_INSTANCE="<your-cloud-sql-instance-id>"
export CLOUD_SQL_MYSQL_DATABASE="<your-database-name>"
export CLOUD_SQL_MYSQL_USER="<your-database-user>"  # Optional
export CLOUD_SQL_MYSQL_PASSWORD="<your-database-password>"  # Optional
export CLOUD_SQL_MYSQL_IP_TYPE="PUBLIC"  # Optional: `PUBLIC`, `PRIVATE`, `PSC`. Defaults to `PUBLIC`.
```

> **Note:** If your instance uses private IPs, you must run the MCP server in the same Virtual Private Cloud (VPC) network.

Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI):

```json
{
  "mcpServers": {
    "cloud-sql-mysql": {
      "command": "toolbox",
      "args": ["--prebuilt", "cloud-sql-mysql", "--stdio"],
      "env": {
        "CLOUD_SQL_MYSQL_PROJECT": "your-project-id",
        "CLOUD_SQL_MYSQL_REGION": "your-region",
        "CLOUD_SQL_MYSQL_INSTANCE": "your-instance-id",
        "CLOUD_SQL_MYSQL_DATABASE": "your-database-name",
        "CLOUD_SQL_MYSQL_USER": "your-username",
        "CLOUD_SQL_MYSQL_PASSWORD": "your-password"
      }
    }
  }
}
```

## Usage

Once configured, the MCP server will automatically provide Cloud SQL for MySQL capabilities to your AI assistant. You can:

*   "Show me the schema for the 'orders' table."
*   "List the top 10 active queries."
*   "Check for tables missing unique indexes."

## Server Capabilities

The Cloud SQL for MySQL MCP server provides the following tools:

| Tool Name | Description |
| :--- | :--- |
| `execute_sql` | Use this tool to execute SQL. |
| `list_active_queries` | Lists top N ongoing queries from processlist and innodb_trx. |
| `get_query_plan` | Provide information about how MySQL executes a SQL statement (EXPLAIN). |
| `list_tables` | Lists detailed schema information for user-created tables. |
| `list_tables_missing_unique_indexes` | Find tables that do not have primary or unique key constraint. |
| `list_table_fragmentation` | List table fragmentation in MySQL. |

## Documentation

For more information, visit the [Cloud SQL for MySQL documentation](https://cloud.google.com/sql/docs/mysql).
