# AlloyDB for PostgreSQL Admin MCP Server

The AlloyDB Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud AlloyDB for PostgreSQL resources. It supports full lifecycle control, from creating clusters and instances to exploring schemas and running queries.

## Features

An editor configured to use the AlloyDB MCP server can use its AI capabilities to help you:

* **Provision & Manage Infrastructure**: Create and manage AlloyDB clusters, instances, and users

To connect to the database to explore and query data, search the MCP store for the AlloyDB for PostgreSQL MCP Server.

## Prerequisites

*   [Node.js](https://nodejs.org/) installed.

> [!NOTE]
> If you encounter issues with Windows Defender blocking the execution, you may need to configure an allowlist. See [Configure exclusions for Microsoft Defender Antivirus](https://learn.microsoft.com/en-us/microsoft-365/security/defender-endpoint/configure-exclusions-microsoft-defender-antivirus?view=o365-worldwide) for more details.

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
      "command": "npx",
      "args": ["-y", "@toolbox-sdk/server", "--prebuilt", "alloydb-postgres-admin", "--stdio"]
    }
  }
}
```

## Documentation

For more information, visit the [AlloyDB for PostgreSQL documentation](https://cloud.google.com/alloydb/docs).
