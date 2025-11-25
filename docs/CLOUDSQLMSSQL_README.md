# Cloud SQL for SQL Server MCP Server

The Cloud SQL for SQL Server Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud SQL for SQL Server databases. It supports connecting to instances, exploring schemas, and running queries.

## Features

An editor configured to use the Cloud SQL for SQL Server MCP server can use its AI capabilities to help you:

- **Query Data** - Execute SQL queries
- **Explore Schema** - List tables and view schema details

For Cloud SQL infrastructure management, search the MCP store for the Cloud SQL for SQL Server Admin MCP Server.

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

*   A Google Cloud project with the **Cloud SQL Admin API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   Cloud SQL Client (`roles/cloudsql.client`)

> **Note:** If your instance uses private IPs, you must run the MCP server in the same Virtual Private Cloud (VPC) network.

## Install & Configuration

1. In the Antigravity MCP Store, click the "Install" button.

2. Add the required inputs for your [instance](https://cloud.google.com/sql/docs/sqlserver/instance-info) in the configuration pop-up, then click "Save". You can update this configuration at any time in the "Configure" tab.

You'll now be able to see all enabled tools in the "Tools" tab.

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

## Custom MCP Server Configuration

The MCP server is configured using environment variables.

```bash
export CLOUD_SQL_MSSQL_PROJECT="<your-gcp-project-id>"
export CLOUD_SQL_MSSQL_REGION="<your-cloud-sql-region>"
export CLOUD_SQL_MSSQL_INSTANCE="<your-cloud-sql-instance-id>"
export CLOUD_SQL_MSSQL_DATABASE="<your-database-name>"
export CLOUD_SQL_MSSQL_USER="<your-database-user>"  # Optional
export CLOUD_SQL_MSSQL_PASSWORD="<your-database-password>"  # Optional
export CLOUD_SQL_MSSQL_IP_TYPE="PUBLIC"  # Optional: `PUBLIC`, `PRIVATE`, `PSC`. Defaults to `PUBLIC`.
```


Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI, `mcp_config.json` for Antigravity):

```json
{
  "mcpServers": {
    "cloud-sql-mssql": {
      "command": "toolbox",
      "args": ["--prebuilt", "cloud-sql-mssql", "--stdio"],
      "env": {
        "CLOUD_SQL_MSSQL_PROJECT": "your-project-id",
        "CLOUD_SQL_MSSQL_REGION": "your-region",
        "CLOUD_SQL_MSSQL_INSTANCE": "your-instance-id",
        "CLOUD_SQL_MSSQL_DATABASE": "your-database-name",
        "CLOUD_SQL_MSSQL_USER": "your-username",
        "CLOUD_SQL_MSSQL_PASSWORD": "your-password"
      }
    }
  }
}
```

## Documentation

For more information, visit the [Cloud SQL for SQL Server documentation](https://cloud.google.com/sql/docs/sqlserver).
