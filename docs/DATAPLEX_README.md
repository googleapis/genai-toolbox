# Dataplex MCP Server

The Dataplex Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud Dataplex Catalog. It supports searching and looking up entries and aspect types.

## Features

An editor configured to use the Dataplex MCP server can use its AI capabilities to help you:

- **Search Catalog** - Search for entries in Dataplex Catalog
- **Explore Metadata** - Lookup specific entries and search aspect types

## Prerequisites

*   [Node.js](https://nodejs.org/) installed.
*   A Google Cloud project with the **Dataplex API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   Dataplex Viewer (`roles/dataplex.viewer`) or equivalent permissions to read catalog entries.

## Install & Configuration

1. In the Antigravity MCP Store, click the "Install" button.

2. Add the required inputs in the configuration pop-up, then click "Save". You can update this configuration at any time in the "Configure" tab.

You'll now be able to see all enabled tools in the "Tools" tab.

> [!NOTE]
> If you encounter issues with Windows Defender blocking the execution, you may need to configure an allowlist. See [Configure exclusions for Microsoft Defender Antivirus](https://learn.microsoft.com/en-us/microsoft-365/security/defender-endpoint/configure-exclusions-microsoft-defender-antivirus?view=o365-worldwide) for more details.

## Usage

Once configured, the MCP server will automatically provide Dataplex capabilities to your AI assistant. You can:

*   "Search for entries related to 'sales' in Dataplex."
*   "Look up details for the entry 'projects/my-project/locations/us-central1/entryGroups/my-group/entries/my-entry'."

## Server Capabilities

The Dataplex MCP server provides the following tools:

| Tool Name             | Description                                      |
|:----------------------|:-------------------------------------------------|
| `search_entries`      | Search for entries in Dataplex Catalog.          |
| `lookup_entry`        | Retrieve a specific entry from Dataplex Catalog. |
| `search_aspect_types` | Find aspect types relevant to the query.         |

## Custom MCP Server Configuration

The MCP server is configured using environment variables.

```bash
export DATAPLEX_PROJECT="<your-gcp-project-id>"
```

Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI, `mcp_config.json` for Antigravity):

```json
{
  "mcpServers": {
    "dataplex": {
      "command": "npx",
      "args": ["-y", "@toolbox-sdk/server", "--prebuilt", "dataplex", "--stdio"],
      "env": {
        "DATAPLEX_PROJECT": "your-project-id"
      }
    }
  }
}
```

## Documentation

For more information, visit the [Dataplex documentation](https://cloud.google.com/dataplex/docs).
