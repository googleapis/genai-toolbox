# Cloud SQL for SQL Server MCP Server

The Cloud SQL for SQL Server Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud SQL for SQL Server databases. It supports connecting to instances, exploring schemas, and running queries.

## Features

An editor configured to use the Cloud SQL for SQL Server MCP server can use its AI capabilities to help you:

- **Query Data** - Execute SQL queries
- **Explore Schema** - List tables and view schema details

## Installation and Setup

### Prerequisites

*   A Google Cloud project with the **Cloud SQL Admin API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   Cloud SQL Client (`roles/cloudsql.client`)

### Configuration

The MCP server is configured using environment variables.

```bash
export CLOUD_SQL_MSSQL_PROJECT="<your-gcp-project-id>"
export CLOUD_SQL_MSSQL_REGION="<your-cloud-sql-region>"
export CLOUD_SQL_MSSQL_INSTANCE="<your-cloud-sql-instance-id>"
export CLOUD_SQL_MSSQL_DATABASE="<your-database-name>"
export CLOUD_SQL_MSSQL_USER="<your-database-user>"  # Optional
export CLOUD_SQL_MSSQL_PASSWORD="<your-database-password>"  # Optional
export CLOUD_SQL_MSSQL_IP_TYPE="PUBLIC"  # Optional: `PUBLIC`, `PRIVATE`, `PSC`. Defaults to `PUBLIC`.
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
        "cloud-sql-mssql": {
          "command": "docker",
          "args": [
            "run",
            "-i",
            "--rm",
            "-e",
            "CLOUD_SQL_MSSQL_PROJECT",
            "-e",
            "CLOUD_SQL_MSSQL_REGION",
            "-e",
            "CLOUD_SQL_MSSQL_INSTANCE",
            "-e",
            "CLOUD_SQL_MSSQL_DATABASE",
            "-e",
            "CLOUD_SQL_MSSQL_USER",
            "-e",
            "CLOUD_SQL_MSSQL_PASSWORD",
            "-e",
            "CLOUD_SQL_MSSQL_IP_TYPE",
            "-e",
            "GOOGLE_APPLICATION_CREDENTIALS=/tmp/keys/adc.json",
            "-v",
            "${GOOGLE_APPLICATION_CREDENTIAL}:/tmp/keys/adc.json:ro",
            "us-central1-docker.pkg.dev/database-toolbox/toolbox/toolbox:latest",
            "--prebuilt",
            "cloud-sql-mssql",
            "--stdio"
          ],
          "env": {
            "CLOUD_SQL_MSSQL_PROJECT": "your-project-id",
            "CLOUD_SQL_MSSQL_REGION": "your-region",
            "CLOUD_SQL_MSSQL_INSTANCE": "your-instance-id",
            "CLOUD_SQL_MSSQL_DATABASE": "your-database-name",
            "CLOUD_SQL_MSSQL_USER": "your-username",
            "CLOUD_SQL_MSSQL_PASSWORD": "your-password"
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
        "cloud-sql-mssql": {
          "command": "./path/to/toolbox",
          "args": ["--prebuilt", "cloud-sql-mssql", "--stdio"],
          "env": {
            "CLOUD_SQL_MSSQL_PROJECT": "your-project-id",
            "CLOUD_SQL_MSSQL_REGION": "your-region",
            "CLOUD_SQL_MSSQL_INSTANCE": "your-instance-id",
            "CLOUD_SQL_MSSQL_DATABASE": "your-database-name",
            "CLOUD_SQL_MSSQL_USER": "your-username",
            "CLOUD_SQL_MSSQL_PASSWORD": "your-password"
          }
        }
      }
    }
    ```

## Usage

Once configured, the MCP server will automatically provide Cloud SQL for SQL Server capabilities to your AI assistant. You can:

*   "Select top 10 rows from the customers table."
*   "List all tables in the database."

## Server Capabilities

The Cloud SQL for SQL Server MCP server provides the following tools:

| Tool Name | Description |
| :--- | :--- |
| `execute_sql` | Use this tool to execute SQL. |
| `list_tables` | Lists detailed schema information for user-created tables. |

## Documentation

For more information, visit the [Cloud SQL for SQL Server documentation](https://cloud.google.com/sql/docs/sqlserver).
