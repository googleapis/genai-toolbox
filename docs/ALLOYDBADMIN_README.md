# AlloyDB for PostgreSQL Admin MCP Server

The AlloyDB Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud AlloyDB for PostgreSQL resources. It supports full lifecycle control, from creating clusters and instances to exploring schemas and running queries.

## Features

An editor configured to use the AlloyDB MCP server can use its AI capabilities to help you:

* ***Provision & Manage Infrastructure**: Create and manage AlloyDB clusters, instances, and users

## Installation and Setup

### Prerequisites

*   Download and install [MCP Toolbox](https://github.com/googleapis/genai-toolbox)
  1.  **Download the Toolbox binary**:
      Download the latest binary for your operating system and architecture from the storage bucket:
      `https://storage.googleapis.com/genai-toolbox/v0.20.0/<os>/<arch>/toolbox`
      *   Replace `<os>` with `linux`, `darwin` (macOS), or `windows`.
      *   Replace `<arch>` with `amd64` (Intel) or `arm64` (Apple Silicon).
      Check the [releases page](https://github.com/googleapis/genai-toolbox/releases) for OS and CPU architecture support.
      

  2.  **Make it executable**:
      ```bash
      chmod +x toolbox
      ```

  3.  **Add the binary to $PATH in `.~/bash_profile`**:
      ```bash
      export PATH=$PATH:/path/to/toolbox
      ```
    
    You may need to restart Antigravity for changes to take effect.

*   A Google Cloud project with the **AlloyDB API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   AlloyDB Admin (`roles/alloydb.admin`) (for managing infrastructure)
    *   Service Usage Consumer (`roles/serviceusage.serviceUsageConsumer`)

### Configuration

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
