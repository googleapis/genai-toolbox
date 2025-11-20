# AlloyDB for PostgreSQL Admin MCP Server

The AlloyDB Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud AlloyDB for PostgreSQL resources. It supports full lifecycle control, from creating clusters and instances to exploring schemas and running queries.

## Features

An editor configured to use the AlloyDB MCP server can use its AI capabilities to help you:

- **Provision & Manage Infrastructure** - Create and manage AlloyDB clusters, instances, and users

## Installation and Setup

### Prerequisites

*   A Google Cloud project with the **AlloyDB API** enabled.
*   Ensure the `GOOGLE_APPLICATION_CREDENTIALS` environment variable is set with [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud).
*   IAM Permissions:
    *   AlloyDB Admin (`roles/alloydb.admin`) (for managing infrastructure)
    *   Service Usage Consumer (`roles/serviceusage.serviceUsageConsumer`)

### Configuration

#### Docker Configuration

1.  **Install [Docker](https://docs.docker.com/install/)**.

2.  **Configure your client**:
    Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI):

    ```json
    {
      "mcpServers": {
        "alloydb-admin": {
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
            "alloydb-postgres-admin",
            "--stdio"
          ],
          "env": {}
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
        "alloydb-admin": {
          "command": "./path/to/toolbox",
          "args": ["--prebuilt", "alloydb-postgres-admin", "--stdio"],
        }
      }
    }
    ```

## Usage

Once configured, the MCP server will automatically provide AlloyDB capabilities to your AI assistant. You can:

*   "Create a new AlloyDB cluster named 'e-commerce-prod' in the 'my-gcp-project' project."
*   "Add a read-only instance to my 'e-commerce-prod' cluster."
*   "Create a new user named 'analyst' with read access to all tables."

## Server Capabilities

The AlloyDB MCP server provides the following tools:

| Tool Name | Description |
| :--- | :--- |
| `create_cluster` | Create an AlloyDB cluster. |
| `create_instance` | Create an AlloyDB instance (PRIMARY, READ-POOL, or SECONDARY). |
| `create_user` | Create ALLOYDB-BUILT-IN or IAM-based users for an AlloyDB cluster. |
| `get_cluster` | Get details about an AlloyDB cluster. |
| `get_instance` | Get details about an AlloyDB instance. |
| `get_user` | Get details about a user in an AlloyDB cluster. |
| `list_clusters` | List clusters in a given project and location. |
| `list_instances` | List instances in a given project and location. |
| `list_users` | List users in a given project and location. |
| `wait_for_operation` | Poll the operations API until the operation is done. |


## Documentation

For more information, visit the [AlloyDB for PostgreSQL documentation](https://cloud.google.com/alloydb/docs).
