# Cloud Spanner Admin MCP Server

The Cloud Spanner Admin Model Context Protocol (MCP) Server gives AI-powered development tools the ability to manage your Google Cloud Spanner infrastructure. It supports creating instances.

## Features

An editor configured to use the Cloud Spanner Admin MCP server can use its AI capabilities to help you:

- **Provision & Manage Infrastructure** - Create Cloud Spanner instances

## Prerequisites

*   [Node.js](https://nodejs.org/) installed.
*   A Google Cloud project with the **Cloud Spanner Admin API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   Cloud Spanner Admin (`roles/spanner.admin`)

## Install & Configuration

In the Antigravity MCP Store, click the "Install" button.

You'll now be able to see all enabled tools in the "Tools" tab.

> [!NOTE]
> If you encounter issues with Windows Defender blocking the execution, you may need to configure an allowlist. See [Configure exclusions for Microsoft Defender Antivirus](https://learn.microsoft.com/en-us/microsoft-365/security/defender-endpoint/configure-exclusions-microsoft-defender-antivirus?view=o365-worldwide) for more details.

## Usage

Once configured, the MCP server will automatically provide Cloud Spanner Admin capabilities to your AI assistant. You can:

   * "Create a new Spanner instance named 'my-spanner-instance' in the 'my-gcp-project' project with config 'regional-us-central1', edition 'ENTERPRISE', and 1 node."

## Server Capabilities

The Cloud Spanner Admin MCP server provides the following tools:

| Tool Name         | Description                      |
|:------------------|:---------------------------------|
| `create_instance` | Create a Cloud Spanner instance. |

## Custom MCP Server Configuration

Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI, `mcp_config.json` for Antigravity):

```json
{
  "mcpServers": {
    "spanner-admin": {
      "command": "npx",
      "args": ["-y", "@toolbox-sdk/server", "--prebuilt", "spanner-admin", "--stdio"]
    }
  }
}
```

## Documentation

For more information, visit the [Cloud Spanner Admin API documentation](https://cloud.google.com/spanner/docs/reference/rpc/google.spanner.admin.instance.v1).
