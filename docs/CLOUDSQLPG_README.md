# Cloud SQL for PostgreSQL MCP Server

The Cloud SQL for PostgreSQL Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud SQL for PostgreSQL databases. It supports connecting to instances, exploring schemas, running queries, and analyzing performance.

## Features

An editor configured to use the Cloud SQL for PostgreSQL MCP server can use its AI capabilities to help you:

- **Query Data** - Execute SQL queries and analyze query plans
- **Explore Schema** - List tables, views, indexes, and triggers
- **Monitor Performance** - View active queries, bloat, and memory configurations
- **Manage Extensions** - List available and installed extensions

## Prerequisites

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

        **On Windows, move binary to the `WindowsApps\` folder**:
        ```
        move "C:\Users\<path-to-binary>\toolbox.exe" "C:\Users\<username>\AppData\Local\Microsoft\WindowsApps\"
        ```
    
        **Tip:** Ensure the destination folder for your binary is included in
        your system's PATH environment variable. To check `PATH`, use `echo
        $PATH` (or `echo %PATH%` on Windows).

        **Note:** You may need to restart Antigravity for changes to take effect.

*   A Google Cloud project with the **Cloud SQL Admin API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   Cloud SQL Client (`roles/cloudsql.client`)

> **Note:** If your instance uses private IPs, you must run the MCP server in the same Virtual Private Cloud (VPC) network.

## Install & Configuration

1. In the Antigravity MCP Store, click the "Install" button.

2. Add the required inputs for your [instance](https://cloud.google.com/sql/docs/postgres/instance-info) in the configuration pop-up, then click "Save". You can update this configuration at anytime in the "Configure" tab.

You'll now be able to see all enabled tools in the "Tools" tab.

## Usage

Once configured, the MCP server will automatically provide Cloud SQL for PostgreSQL capabilities to your AI assistant. You can:

*   "Show me the top 5 bloated tables."
*   "List all installed extensions."
*   "Explain the query plan for SELECT * FROM users."

## Server Capabilities

The Cloud SQL for PostgreSQL MCP server provides the following tools:

| Tool Name                        | Description                                                    |
|:---------------------------------|:---------------------------------------------------------------|
| `execute_sql`                    | Use this tool to execute sql.                                  |
| `list_tables`                    | Lists detailed schema information for user-created tables.     |
| `list_active_queries`            | List the top N currently running queries.                      |
| `list_available_extensions`      | Discover all PostgreSQL extensions available for installation. |
| `list_installed_extensions`      | List all installed PostgreSQL extensions.                      |
| `list_autovacuum_configurations` | List PostgreSQL autovacuum-related configurations.             |
| `list_memory_configurations`     | List PostgreSQL memory-related configurations.                 |
| `list_top_bloated_tables`        | List the top tables by dead-tuple (approximate bloat signal).  |
| `list_replication_slots`         | List key details for all PostgreSQL replication slots.         |
| `list_invalid_indexes`           | Lists all invalid PostgreSQL indexes.                          |
| `get_query_plan`                 | Generate a PostgreSQL EXPLAIN plan in JSON format.             |
| `list_views`                     | Lists views in the database.                                   |
| `list_schemas`                   | Lists all schemas in the database.                             |
| `database_overview`              | Fetches the current state of the PostgreSQL server.            |
| `list_triggers`                  | Lists all non-internal triggers in a database.                 |
| `list_indexes`                   | Lists available user indexes in the database.                  |
| `list_sequences`                 | Lists sequences in the database.                               |

## Documentation

For more information, visit the [Cloud SQL for PostgreSQL documentation](https://cloud.google.com/sql/docs/postgres).
