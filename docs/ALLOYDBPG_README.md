# AlloyDB for PostgreSQL MCP Server

The AlloyDB Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud AlloyDB for PostgreSQL resources. It supports full lifecycle control, from exploring schemas and running queries to monitoring your database.

## Features

An editor configured to use the AlloyDB MCP server can use its AI capabilities to help you:

- **Explore Schemas and Data** - List tables, get table details, and view data
- **Execute SQL** - Run SQL queries directly from your editor
- **Monitor Performance** - View active queries, query plans, and other performance metrics (via observability tools)
- **Manage Extensions** - List available and installed PostgreSQL extensions

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

*   A Google Cloud project with the **AlloyDB API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   AlloyDB Client (`roles/alloydb.client`) (for connecting and querying)
    *   Service Usage Consumer (`roles/serviceusage.serviceUsageConsumer`)

### Configuration

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

> **Note:** If your AlloyDB instance uses private IPs, you must run the MCP server in the same Virtual Private Cloud (VPC) network.

Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI):

```json
{
  "mcpServers": {
    "alloydb-postgres": {
      "command": "toolbox",
      "args": ["--prebuilt", "alloydb-postgres", "--stdio"],
    }
  }
}
```

## Usage

Once configured, the MCP server will automatically provide AlloyDB capabilities to your AI assistant. You can:

*   "Show me all tables in the 'orders' database."
*   "What are the columns in the 'products' table?"
*   "How many orders were placed in the last 30 days?"

## Server Capabilities

The AlloyDB MCP server provides the following tools:

| Tool Name | Description |
| :--- | :--- |
| `list_tables` | Lists detailed schema information for user-created tables. |
| `execute_sql` | Executes a SQL query. |
| `list_active_queries` | List currently running queries. |
| `list_available_extensions` | List available extensions for installation. |
| `list_installed_extensions` | List installed extensions. |
| `get_query_plan` | Get query plan for a SQL statement. |
| `list_autovacuum_configurations` | List autovacuum configurations and their values. |
| `list_memory_configurations` | List memory configurations and their values. |
| `list_top_bloated_tables` | List top bloated tables. |
| `list_replication_slots` | List replication slots. |
| `list_invalid_indexes` | List invalid indexes. |

## Documentation

For more information, visit the [AlloyDB for PostgreSQL documentation](https://cloud.google.com/alloydb/docs).
