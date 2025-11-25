# AlloyDB for PostgreSQL Admin MCP Server

The AlloyDB Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud AlloyDB for PostgreSQL resources. It supports full lifecycle control, from creating clusters and instances to exploring schemas and running queries.

## Features

An editor configured to use the AlloyDB MCP server can use its AI capabilities to help you:

* **Provision & Manage Infrastructure**: Create and manage AlloyDB clusters, instances, and users

To connect to the database to explore and query data, search the MCP store for the AlloyDB for PostgreSQL MCP Server.

## Prerequisites

*   Download and install [MCP Toolbox](https://github.com/googleapis/genai-toolbox):
    1.  **Download the Toolbox binary**:
        Download the latest binary for your operating system and architecture from the storage bucket. Check the [releases page](https://github.com/googleapis/genai-toolbox/releases) for additional versions: 
      
        <!-- {x-release-please-start-version} -->
        * To install Toolbox as a binary on Linux (AMD64):
          ```bash
          curl -L -o toolbox https://storage.googleapis.com/genai-toolbox/v0.21.0/linux/amd64/toolbox
          ```

        * To install Toolbox as a binary on macOS (Apple Silicon):
          ```bash
          curl -L -o toolbox https://storage.googleapis.com/genai-toolbox/v0.21.0/darwin/arm64/toolbox
          ```

        * To install Toolbox as a binary on macOS (Intel):
          ```bash
          curl -L -o toolbox https://storage.googleapis.com/genai-toolbox/v0.21.0/darwin/amd64/toolbox
          ```

        * To install Toolbox as a binary on Windows (AMD64):
          ```powershell
          curl -o toolbox.exe "https://storage.googleapis.com/genai-toolbox/v0.21.0/windows/amd64/toolbox.exe"
          ```
        <!-- {x-release-please-end} -->
        
    2.  **Make it executable**:

        ```bash
        chmod +x toolbox
        ```

    3.  **Add the binary to $PATH in `.~/bash_profile`** (Note: You may need to restart Antigravity for changes to take effect.):

        ```bash
        export PATH=$PATH:path/to/folder
        ```

        **On Windows, move binary to the `WindowsApps\` folder**:
        ```
        move "C:\Users\<path-to-binary>\toolbox.exe" "C:\Users\<username>\AppData\Local\Microsoft\WindowsApps\"
        ```
    
        **Tip:** Ensure the destination folder for your binary is included in
        your system's PATH environment variable. To check `PATH`, use `echo
        $PATH` (or `echo %PATH%` on Windows).

*   A Google Cloud project with the **AlloyDB API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   AlloyDB Admin (`roles/alloydb.admin`) (for managing infrastructure)
    *   Service Usage Consumer (`roles/serviceusage.serviceUsageConsumer`)

## Install & Configuration

1. In the Antigravity MCP Store, click the "Install" button.

You'll now be able to see all enabled tools in the "Tools" tab.

## Usage

Once configured, the MCP server will automatically provide AlloyDB capabilities to your AI assistant. You can:

*   "Create a new AlloyDB cluster named 'e-commerce-prod' in the 'my-gcp-project' project."
*   "Add a read-only instance to my 'e-commerce-prod' cluster."
*   "Create a new user named 'analyst' with read access to all tables."

## Server Capabilities

The AlloyDB MCP server provides the following tools:

| Tool Name            | Description                                                        |
|:---------------------|:-------------------------------------------------------------------|
| `create_cluster`     | Create an AlloyDB cluster.                                         |
| `create_instance`    | Create an AlloyDB instance (PRIMARY, READ-POOL, or SECONDARY).     |
| `create_user`        | Create ALLOYDB-BUILT-IN or IAM-based users for an AlloyDB cluster. |
| `get_cluster`        | Get details about an AlloyDB cluster.                              |
| `get_instance`       | Get details about an AlloyDB instance.                             |
| `get_user`           | Get details about a user in an AlloyDB cluster.                    |
| `list_clusters`      | List clusters in a given project and location.                     |
| `list_instances`     | List instances in a given project and location.                    |
| `list_users`         | List users in a given project and location.                        |
| `wait_for_operation` | Poll the operations API until the operation is done.               |

## Custom MCP Server Configuration

Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI, `mcp_config.json` for Antigravity):

```json
{
  "mcpServers": {
    "alloydb-admin": {
      "command": "toolbox",
      "args": ["--prebuilt", "alloydb-postgres-admin", "--stdio"]
    }
  }
}
```

## Documentation

For more information, visit the [AlloyDB for PostgreSQL documentation](https://cloud.google.com/alloydb/docs).
