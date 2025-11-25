# AlloyDB for PostgreSQL MCP Server

The AlloyDB Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud AlloyDB for PostgreSQL resources. It supports full lifecycle control, from exploring schemas and running queries to monitoring your database.

## Features

An editor configured to use the AlloyDB MCP server can use its AI capabilities to help you:

- **Explore Schemas and Data** - List tables, get table details, and view data
- **Execute SQL** - Run SQL queries directly from your editor
- **Monitor Performance** - View active queries, query plans, and other performance metrics (via observability tools)
- **Manage Extensions** - List available and installed PostgreSQL extensions

For AlloyDB infrastructure management, search the MCP store for the AlloyDB for PostgreSQL Admin MCP Server.

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
    *   AlloyDB Client (`roles/alloydb.client`) (for connecting and querying)
    *   Service Usage Consumer (`roles/serviceusage.serviceUsageConsumer`)

> **Note:** If your AlloyDB instance uses private IPs, you must run the MCP server in the same Virtual Private Cloud (VPC) network.

## Install & Configuration

1. In the Antigravity MCP Store, click the "Install" button.

2. Add the required inputs for your [cluster](https://docs.cloud.google.com/alloydb/docs/cluster-list) in the configuration pop-up, then click "Save". You can update this configuration at any time in the "Configure" tab.

You'll now be able to see all enabled tools in the "Tools" tab.

## Usage

Once configured, the MCP server will automatically provide AlloyDB capabilities to your AI assistant. You can:

*   "Show me all tables in the 'orders' database."
*   "What are the columns in the 'products' table?"
*   "How many orders were placed in the last 30 days?"

## Server Capabilities

The AlloyDB MCP server provides the following tools:

| Tool Name                        | Description                                                |
|:---------------------------------|:-----------------------------------------------------------|
| `list_tables`                    | Lists detailed schema information for user-created tables. |
| `execute_sql`                    | Executes a SQL query.                                      |
| `list_active_queries`            | List currently running queries.                            |
| `list_available_extensions`      | List available extensions for installation.                |
| `list_installed_extensions`      | List installed extensions.                                 |
| `get_query_plan`                 | Get query plan for a SQL statement.                        |
| `list_autovacuum_configurations` | List autovacuum configurations and their values.           |
| `list_memory_configurations`     | List memory configurations and their values.               |
| `list_top_bloated_tables`        | List top bloated tables.                                   |
| `list_replication_slots`         | List replication slots.                                    |
| `list_invalid_indexes`           | List invalid indexes.                                      |

## Custom MCP Server Configuration

The AlloyDB MCP server is configured using environment variables.

```bash
export ALLOYDB_POSTGRES_PROJECT="<your-gcp-project-id>"
export ALLOYDB_POSTGRES_REGION="<your-alloydb-region>"
export ALLOYDB_POSTGRES_CLUSTER="<your-alloydb-cluster-id>"
export ALLOYDB_POSTGRES_INSTANCE="<your-alloydb-instance-id>"
export ALLOYDB_POSTGRES_DATABASE="<your-database-name>"
export ALLOYDB_POSTGRES_USER="<your-database-user>"  # Optional
export ALLOYDB_POSTGRES_PASSWORD="<your-database-password>"  # Optional
export ALLOYDB_POSTGRES_IP_TYPE="PUBLIC"  # Optional: `PUBLIC`, `PRIVATE`, `PSC`. Defaults to `PUBLIC`.
```

Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI, `mcp_config.json` for Antigravity):

```json
{
  "mcpServers": {
    "alloydb-postgres": {
      "command": "toolbox",
      "args": ["--prebuilt", "alloydb-postgres", "--stdio"]
    }
  }
}
```

## Documentation

For more information, visit the [AlloyDB for PostgreSQL documentation](https://cloud.google.com/alloydb/docs).
