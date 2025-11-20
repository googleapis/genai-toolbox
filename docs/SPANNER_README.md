# Cloud Spanner MCP Server

The Cloud Spanner Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud Spanner databases. It supports executing SQL queries and exploring schemas.

## Features

An editor configured to use the Cloud Spanner MCP server can use its AI capabilities to help you:

- **Query Data** - Execute DML and DQL SQL queries
- **Explore Schema** - List tables and view schema details

## Installation and Setup

### Prerequisites

*   A Google Cloud project with the **Cloud Spanner API** enabled.
*   Ensure the `GOOGLE_APPLICATION_CREDENTIALS` environment variable is set with [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud).
*   IAM Permissions:
    *   Cloud Spanner Database User (`roles/spanner.databaseUser`) (for data access)
    *   Cloud Spanner Viewer (`roles/spanner.viewer`) (for schema access)

### Configuration

#### Docker Configuration

1.  **Install [Docker](https://docs.docker.com/install/)**.

2.  **Configure your client**:
    Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI):

    ```json
    {
      "mcpServers": {
        "spanner": {
          "command": "docker",
          "args": [
            "run",
            "-i",
            "--rm",
            "-e",
            "GOOGLE_APPLICATION_CREDENTIALS=/tmp/keys/adc.json",
            "-v",
            "${GOOGLE_APPLICATION_CREDENTIAL}:/tmp/keys/adc.json:ro",
            "us-central1-docker.pkg.dev/database-toolbox/toolbox/toolbox:latest",
            "--prebuilt",
            "spanner",
            "--stdio"
          ],
          "env": {
            "SPANNER_PROJECT": "your-project-id",
            "SPANNER_INSTANCE": "your-instance-id",
            "SPANNER_DATABASE": "your-database-name",
            "SPANNER_DIALECT": "googlesql"
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
        "spanner": {
          "command": "./path/to/toolbox",
          "args": ["--prebuilt", "spanner", "--stdio"],
          "env": {
            "SPANNER_PROJECT": "your-project-id",
            "SPANNER_INSTANCE": "your-instance-id",
            "SPANNER_DATABASE": "your-database-name",
            "SPANNER_DIALECT": "googlesql"
          }
        }
      }
    }
    ```

## Usage

Once configured, the MCP server will automatically provide Cloud Spanner capabilities to your AI assistant. You can:

*   "Execute a DML query to update customer names."
*   "List all tables in the `my-database`."
*   "Execute a DQL query to select data from `orders` table."

## Server Capabilities

The Cloud Spanner MCP server provides the following tools:

| Tool Name | Description |
| :--- | :--- |
| `execute_sql` | Use this tool to execute DML SQL. |
| `execute_sql_dql` | Use this tool to execute DQL SQL. |
| `list_tables` | Lists detailed schema information for user-created tables. |

## Documentation

For more information, visit the [Cloud Spanner documentation](https://cloud.google.com/spanner/docs).
