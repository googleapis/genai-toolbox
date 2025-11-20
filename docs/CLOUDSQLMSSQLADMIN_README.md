# Cloud SQL for SQL Server Admin MCP Server

The Cloud SQL for SQL Server Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud SQL for SQL Server databases. It supports connecting to instances, exploring schemas, and running queries.

## Features

An editor configured to use the Cloud SQL for SQL Server MCP server can use its AI capabilities to help you:

- **Provision & Manage Infrastructure** - Create and manage Cloud SQL instances and users

## Installation and Setup

### Prerequisites

*   A Google Cloud project with the **Cloud SQL Admin API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
  * Cloud SQL Viewer (`roles/cloudsql.viewer`)
  * Cloud SQL Admin (`roles/cloudsql.admin`)

### Configuration

#### Docker Configuration

1.  **Install [Docker](https://docs.docker.com/install/)**.
2.  Ensure the `GOOGLE_APPLICATION_CREDENTIALS` environment variable is set with [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud).

3.  **Configure your client**:
    Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI):

    ```json
    {
      "mcpServers": {
        "cloud-sql-sqlserver-admin": {
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
            "cloud-sql-mssql-admin",
            "--stdio"
          ],
          
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
        "cloud-sql-sqlserver-admin": {
          "command": "./path/to/toolbox",
          "args": ["--prebuilt", "cloud-sql-mssql-admin", "--stdio"],
        }
      }
    }
    ```

## Usage

Once configured, the MCP server will automatically provide Cloud SQL for SQL Server capabilities to your AI assistant. You can:

  * "Create a new Cloud SQL for SQL Server instance named 'e-commerce-prod' in the 'my-gcp-project' project."
  * "Create a new user named 'analyst' with read access to all tables."

## Server Capabilities

The Cloud SQL for SQL Server MCP server provides the following tools:

| Tool Name | Description |
| :--- | :--- |
| `create_instance` | Create an instance (PRIMARY, READ-POOL, or SECONDARY). |
| `create_user` | Create BUILT-IN or IAM-based users for an instance. |
| `get_instance` | Get details about an instance. |
| `get_user` | Get details about a user in an instance. |
| `list_instances` | List instances in a given project and location. |
| `list_users` | List users in a given project and location. |
| `wait_for_operation` | Poll the operations API until the operation is done. |

## Documentation

For more information, visit the [Cloud SQL for SQL Server documentation](https://cloud.google.com/sql/docs/sqlserver).
