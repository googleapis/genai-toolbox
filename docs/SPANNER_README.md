# Cloud Spanner MCP Server

The Cloud Spanner Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud Spanner databases. It supports executing SQL queries and exploring schemas.

## Features

An editor configured to use the Cloud Spanner MCP server can use its AI capabilities to help you:

- **Query Data** - Execute DML and DQL SQL queries
- **Explore Schema** - List tables and view schema details

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

*   A Google Cloud project with the **Cloud Spanner API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   Cloud Spanner Database User (`roles/spanner.databaseUser`) (for data access)
    *   Cloud Spanner Viewer (`roles/spanner.viewer`) (for schema access)

## Install & Configuration

1. In the Antigravity MCP Store, click the "Install" button.

2. Add the required inputs for your [instance](https://docs.cloud.google.com/spanner/docs/instances) in the configuration pop-up, then click "Save". You can update this configuration at any time in the "Configure" tab.

You'll now be able to see all enabled tools in the "Tools" tab.

## Usage

Once configured, the MCP server will automatically provide Cloud Spanner capabilities to your AI assistant. You can:

*   "Execute a DML query to update customer names."
*   "List all tables in the `my-database`."
*   "Execute a DQL query to select data from `orders` table."

## Server Capabilities

The Cloud Spanner MCP server provides the following tools:

| Tool Name         | Description                                                |
|:------------------|:-----------------------------------------------------------|
| `execute_sql`     | Use this tool to execute DML SQL.                          |
| `execute_sql_dql` | Use this tool to execute DQL SQL.                          |
| `list_tables`     | Lists detailed schema information for user-created tables. |

## Custom MCP Server Configuration

The MCP server is configured using environment variables.

```bash
export SPANNER_PROJECT="<your-gcp-project-id>"
export SPANNER_INSTANCE="<your-spanner-instance-id>"
export SPANNER_DATABASE="<your-spanner-database-id>"
export SPANNER_DIALECT="googlesql" # Optional: "googlesql" or "postgresql". Defaults to "googlesql".
```

Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI, `mcp_config.json` for Antigravity):

```json
{
  "mcpServers": {
    "spanner": {
      "command": "toolbox",
      "args": ["--prebuilt", "spanner", "--stdio"],
      "env": {
        "SPANNER_PROJECT": "your-project-id",
        "SPANNER_INSTANCE": "your-instance-id",
        "SPANNER_DATABASE": "your-database-name",
        "SPANNER_DIALECT": "googlesql"
      }
    }
  }
}
```

## Documentation

For more information, visit the [Cloud Spanner documentation](https://cloud.google.com/spanner/docs).
