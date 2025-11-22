# Cloud SQL for SQL Server MCP Server

The Cloud SQL for SQL Server Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud SQL for SQL Server databases. It supports connecting to instances, exploring schemas, and running queries.

## Features

An editor configured to use the Cloud SQL for SQL Server MCP server can use its AI capabilities to help you:

- **Query Data** - Execute SQL queries
- **Explore Schema** - List tables and view schema details

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

*   A Google Cloud project with the **Cloud SQL Admin API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   Cloud SQL Client (`roles/cloudsql.client`)

### Configuration

1. **Access the Store**: Open the MCP Store panel within the "..." dropdown at the top of the editor's side panel.
2. **Browse and Install**: Search for "Cloud SQL for SQL Server", and click "Install".
3. **Configuration**: The following configuration is needed for the server:
   * Cloud SQL Project ID: The GCP project ID.
   * Cloud SQL Region: The region of your Cloud SQL instance.
   * Cloud SQL Instance ID: The ID of your Cloud SQL instance.
   * Cloud SQL Database Name: The name of the database.
   * Cloud SQL Database User: (Optional) The database username. Defaults to IAM authentication if unspecified.
   * Cloud SQL Database Password: (Optional) The password for the database user. Defaults to IAM authentication if unspecified.
   * Cloud SQL IP Type: (Optional) The IP type i.e. “Public” or “Private”. Defaults to "Public" if unspecified.

> [!NOTE]
> If your instance uses private IPs, you must run the MCP server in the same Virtual Private Cloud (VPC) network.


## Usage

Once configured, the MCP server will automatically provide Cloud SQL for SQL Server capabilities to your AI assistant. You can:

*   "Select top 10 rows from the customers table."
*   "List all tables in the database."

## Server Capabilities

The Cloud SQL for SQL Server MCP server provides the following tools:

| Tool Name     | Description                                                |
|:--------------|:-----------------------------------------------------------|
| `execute_sql` | Use this tool to execute SQL.                              |
| `list_tables` | Lists detailed schema information for user-created tables. |

## Documentation

For more information, visit the [Cloud SQL for SQL Server documentation](https://cloud.google.com/sql/docs/sqlserver).
