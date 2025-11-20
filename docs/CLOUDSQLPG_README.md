# Cloud SQL for PostgreSQL MCP Server

The Cloud SQL for PostgreSQL Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud SQL for PostgreSQL databases. It supports connecting to instances, exploring schemas, running queries, and analyzing performance.

## Features

An editor configured to use the Cloud SQL for PostgreSQL MCP server can use its AI capabilities to help you:

- **Query Data** - Execute SQL queries and analyze query plans
- **Explore Schema** - List tables, views, indexes, and triggers
- **Monitor Performance** - View active queries, bloat, and memory configurations
- **Manage Extensions** - List available and installed extensions

## Installation and Setup

### Prerequisites

*   A Google Cloud project with the **Cloud SQL Admin API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   Cloud SQL Client (`roles/cloudsql.client`)

### Configuration

The MCP server is configured using environment variables.

```bash
export CLOUD_SQL_POSTGRES_PROJECT="<your-gcp-project-id>"
export CLOUD_SQL_POSTGRES_REGION="<your-cloud-sql-region>"
export CLOUD_SQL_POSTGRES_INSTANCE="<your-cloud-sql-instance-id>"
export CLOUD_SQL_POSTGRES_DATABASE="<your-database-name>"
export CLOUD_SQL_POSTGRES_USER="<your-database-user>"  # Optional
export CLOUD_SQL_POSTGRES_PASSWORD="<your-database-password>"  # Optional
export CLOUD_SQL_POSTGRES_IP_TYPE="PUBLIC"  # Optional: `PUBLIC`, `PRIVATE`, `PSC`. Defaults to `PUBLIC`.
```

> **Note:** If your instance uses private IPs, you must run the MCP server in the same Virtual Private Cloud (VPC) network.


#### Docker Configuration

1.  **Install [Docker](https://docs.docker.com/install/)**.

2.  Ensure the `GOOGLE_APPLICATION_CREDENTIALS` environment variable is set with [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud).

3.  **Configure your client**:
    Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI):

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres": {
          "command": "docker",
          "args": [
            "run",
            "-i",
            "--rm",
            "-e",
            "CLOUD_SQL_POSTGRES_PROJECT",
            "-e",
            "CLOUD_SQL_POSTGRES_REGION",
            "-e",
            "CLOUD_SQL_POSTGRES_INSTANCE",
            "-e",
            "CLOUD_SQL_POSTGRES_DATABASE",
            "-e",
            "CLOUD_SQL_POSTGRES_USER",
            "-e",
            "CLOUD_SQL_POSTGRES_PASSWORD",
            "-e",
            "CLOUD_SQL_POSTGRES_IP_TYPE",
            "-e",
            "GOOGLE_APPLICATION_CREDENTIALS=/tmp/keys/adc.json",
            "-v",
            "${GOOGLE_APPLICATION_CREDENTIAL}:/tmp/keys/adc.json:ro",
            "us-central1-docker.pkg.dev/database-toolbox/toolbox/toolbox:latest",
            "--prebuilt",
            "cloud-sql-postgres",
            "--stdio"
          ],
          "env": {
            "CLOUD_SQL_POSTGRES_PROJECT": "your-project-id",
            "CLOUD_SQL_POSTGRES_REGION": "your-region",
            "CLOUD_SQL_POSTGRES_INSTANCE": "your-instance-id",
            "CLOUD_SQL_POSTGRES_DATABASE": "your-database-name",
            "CLOUD_SQL_POSTGRES_USER": "your-username",
            "CLOUD_SQL_POSTGRES_PASSWORD": "your-password"
          }
        }
      }
    }
    ```

#### Binary Configuration

1.  **Download the Toolbox binary**:
    Download the latest binary for your operating system and architecture from the storage bucket:
    `https://storage.googleapis.com/genai-toolbox/v0.20.0/<os>/<arch>/toolbox`
    *   Replace `<os>` with `linux`, `darwin` (macOS), or `windows`.
    *   Replace `<arch>` with `amd64` (Intel) or `arm64` (Apple Silicon).

2.  **Make it executable**:
    ```bash
    chmod +x toolbox
    ```

3.  **Configure your client**:
    Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI):

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres": {
          "command": "./path/to/toolbox",
          "args": ["--prebuilt", "cloud-sql-postgres", "--stdio"],
          "env": {
            "CLOUD_SQL_POSTGRES_PROJECT": "your-project-id",
            "CLOUD_SQL_POSTGRES_REGION": "your-region",
            "CLOUD_SQL_POSTGRES_INSTANCE": "your-instance-id",
            "CLOUD_SQL_POSTGRES_DATABASE": "your-database-name",
            "CLOUD_SQL_POSTGRES_USER": "your-username",
            "CLOUD_SQL_POSTGRES_PASSWORD": "your-password"
          }
        }
      }
    }
    ```

## Usage

Once configured, the MCP server will automatically provide Cloud SQL for PostgreSQL capabilities to your AI assistant. You can:

*   "Show me the top 5 bloated tables."
*   "List all installed extensions."
*   "Explain the query plan for SELECT * FROM users."

## Server Capabilities

The Cloud SQL for PostgreSQL MCP server provides the following tools:

| Tool Name | Description |
| :--- | :--- |
| `execute_sql` | Use this tool to execute sql. |
| `list_tables` | Lists detailed schema information for user-created tables. |
| `list_active_queries` | List the top N currently running queries. |
| `list_available_extensions` | Discover all PostgreSQL extensions available for installation. |
| `list_installed_extensions` | List all installed PostgreSQL extensions. |
| `list_autovacuum_configurations` | List PostgreSQL autovacuum-related configurations. |
| `list_memory_configurations` | List PostgreSQL memory-related configurations. |
| `list_top_bloated_tables` | List the top tables by dead-tuple (approximate bloat signal). |
| `list_replication_slots` | List key details for all PostgreSQL replication slots. |
| `list_invalid_indexes` | Lists all invalid PostgreSQL indexes. |
| `get_query_plan` | Generate a PostgreSQL EXPLAIN plan in JSON format. |
| `list_views` | Lists views in the database. |
| `list_schemas` | Lists all schemas in the database. |
| `database_overview` | Fetches the current state of the PostgreSQL server. |
| `list_triggers` | Lists all non-internal triggers in a database. |
| `list_indexes` | Lists available user indexes in the database. |
| `list_sequences` | Lists sequences in the database. |

## Documentation

For more information, visit the [Cloud SQL for PostgreSQL documentation](https://cloud.google.com/sql/docs/postgres).
