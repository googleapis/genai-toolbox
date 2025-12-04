# Cloud Spanner MCP Server

The Cloud Spanner Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud Spanner databases. It supports executing SQL queries and exploring schemas.

## Features

An editor configured to use the Cloud Spanner MCP server can use its AI capabilities to help you:

- **Query Data** - Execute DML and DQL SQL queries
- **Explore Schema** - List tables and view schema details

## Prerequisites

*   [Node.js](https://nodejs.org/) installed.
*   A Google Cloud project with the **Cloud Spanner API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   Cloud Spanner Database User (`roles/spanner.databaseUser`) (for data access)
    *   Cloud Spanner Viewer (`roles/spanner.viewer`) (for schema access)

## Install & Configuration

1. In the Antigravity MCP Store, click the "Install" button.

2. Add the required inputs for your [instance](https://docs.cloud.google.com/spanner/docs/instances) in the configuration pop-up, then click "Save". You can update this configuration at any time in the "Configure" tab.

You'll now be able to see all enabled tools in the "Tools" tab.

> [!NOTE]
> If you encounter issues with Windows Defender blocking the execution, you may need to configure an allowlist. See [Configure exclusions for Microsoft Defender Antivirus](https://learn.microsoft.com/en-us/microsoft-365/security/defender-endpoint/configure-exclusions-microsoft-defender-antivirus?view=o365-worldwide) for more details.

## Usage

Once configured, the MCP server will automatically provide Cloud Spanner capabilities to your AI assistant. You can:

*   "Execute a DML query to update customer names."
*   "List all tables in the `my-database`."
*   "Execute a DQL query to select data from `orders` table."

## Server Capabilities

The Cloud Spanner MCP server provides the following tools:

| Tool Name         | Description                                                      |
|:------------------|:-----------------------------------------------------------------|
| `execute_sql`     | Use this tool to execute DML SQL.                                |
| `execute_sql_dql` | Use this tool to execute DQL SQL.                                |
| `list_tables`     | Lists detailed schema information for user-created tables.       |
| `list_graphs`     | Lists detailed graph schema information for user-created graphs. |

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
      "command": "npx",
      "args": ["-y", "@toolbox-sdk/server", "--prebuilt", "spanner", "--stdio"],
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
