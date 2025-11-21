# Cloud SQL for MySQL Admin MCP Server

The Cloud SQL for MySQL Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud SQL for MySQL databases. It supports connecting to instances, exploring schemas, and running queries.

## Features

An editor configured to use the Cloud SQL for MySQL MCP server can use its AI capabilities to help you:

- **Provision & Manage Infrastructure** - Create and manage Cloud SQL instances and users

## Installation and Setup

### Prerequisites

*   Download and install [MCP Toolbox](https://github.com/googleapis/genai-toolbox):
  1.  **Download the Toolbox binary**:
      Download the latest binary for your operating system and architecture from the storage bucket. Check the [releases page](https://github.com/googleapis/genai-toolbox/releases) for OS and CPU architecture support:
      `https://storage.googleapis.com/genai-toolbox/v0.21.0/<os>/<arch>/toolbox`
      *   Replace `<os>` with `linux`, `darwin` (macOS), or `windows`.
      *   Replace `<arch>` with `amd64` (Intel) or `arm64` (Apple Silicon).
      
      <!-- {x-release-please-start-version} -->
      ```
      curl -L -o toolbox https://storage.googleapis.com/genai-toolbox/v0.21.0/linux/amd64/toolbox
      ```
      <!-- {x-release-please-end} -->
  2.  **Make it executable**:
      ```bash
      chmod +x toolbox
      ```

  3.  **Add the binary to $PATH in `.~/bash_profile`**:
      ```bash
      export PATH=$PATH:/path/to/toolbox
      ```
    
**Note:** You may need to restart Antigravity for changes to take effect. 
Windows OS users will need to follow one of the Windows-specific methods.

*   A Google Cloud project with the **Cloud SQL Admin API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
  * Cloud SQL Viewer (`roles/cloudsql.viewer`)
  * Cloud SQL Admin (`roles/cloudsql.admin`)

### Configuration

Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI):

```json
{
  "mcpServers": {
    "cloud-sql-mysql-admin": {
      "command": "toolbox",
      "args": ["--prebuilt", "cloud-sql-mysql-admin", "--stdio"]
    }
  }
}
```

## Usage

Once configured, the MCP server will automatically provide Cloud SQL for MySQL capabilities to your AI assistant. You can:

   * "Create a new Cloud SQL for MySQL instance named 'e-commerce-prod' in the 'my-gcp-project' project."
   * "Create a new user named 'analyst' with read access to all tables."

## Server Capabilities

The Cloud SQL for MySQL MCP server provides the following tools:

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

For more information, visit the [Cloud SQL for MySQL documentation](https://cloud.google.com/sql/docs/mysql).
