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

    3.  **Move binary to `/usr/local/bin/` or `/usr/bin/`**:
        ```bash
        sudo mv toolbox /usr/local/bin/
        # sudo mv toolbox /usr/bin/
        ```

        **On Windows, move binary to the `WindowsApps\` folder**:
        ```
        move "C:\Users\<path-to-binary>\toolbox.exe" "C:\Users\<username>\AppData\Local\Microsoft\WindowsApps\"
        ```
    
        **Tip:** Ensure the destination folder for your binary is included in
        your system's PATH environment variable. To check `PATH`, use `echo
        $PATH` (or `echo %PATH%` on Windows).

        **Note:** You may need to restart Antigravity for changes to take effect.

*   A Google Cloud project with the **AlloyDB API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   AlloyDB Client (`roles/alloydb.client`) (for connecting and querying)
    *   Service Usage Consumer (`roles/serviceusage.serviceUsageConsumer`)

### Configuration

1. **Access the Store**: Open the MCP Store panel within the "..." dropdown at the top of the editor's side panel.
2. **Browse and Install**: Search for "AlloyDB for PostgreSQL", and click "Install".
3. **Configuration**: The following configuration is needed for the server:
   * AlloyDB Project ID: The GCP project ID.
   * AlloyDB Region: The region of your AlloyDB instance.
   * AlloyDB Cluster ID: The ID of your AlloyDB cluster.
   * AlloyDB Instance ID: The ID of your AlloyDB instance.
   * AlloyDB Database Name: The name of the database.
   * AlloyDB Database User: (Optional) The database username. Defaults to IAM authentication if unspecified.
   * AlloyDB Database Password: (Optional) The password for the database user. Defaults to IAM authentication if unspecified.
   * AlloyDB IP Type: (Optional) The IP type i.e. “Public” or “Private”. Defaults to "Public" if unspecified.

> [!NOTE]
> If your AlloyDB instance uses private IPs, you must run the MCP server in the same Virtual Private Cloud (VPC) network.

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

## Documentation

For more information, visit the [AlloyDB for PostgreSQL documentation](https://cloud.google.com/alloydb/docs).
