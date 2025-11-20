# Dataplex MCP Server

The Dataplex Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud Dataplex Catalog. It supports searching and looking up entries and aspect types.

## Features

An editor configured to use the Dataplex MCP server can use its AI capabilities to help you:

- **Search Catalog** - Search for entries in Dataplex Catalog
- **Explore Metadata** - Lookup specific entries and search aspect types

## Installation and Setup

### Prerequisites

*   A Google Cloud project with the **Dataplex API** enabled.
*   Ensure the `GOOGLE_APPLICATION_CREDENTIALS` environment variable is set with [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud).
*   IAM Permissions:
    *   Dataplex Viewer (`roles/dataplex.viewer`) or equivalent permissions to read catalog entries.

### Configuration

#### Docker Configuration

1.  **Install [Docker](https://docs.docker.com/install/)**.

2.  **Configure your client**:
    Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI):

    ```json
    {
      "mcpServers": {
        "dataplex": {
          "command": "docker",
          "args": [
            "run",
            "-i",
            "--rm",
            "-e",
            "GOOGLE_APPLICATION_CREDENTIALS=/tmp/keys/adc.json",
            "-v",
            "${GOOGLE_APPLICATION_CREDENTIAL}:/tmp/keys/adc.json:ro",
            "us-central1-docker.pkg.dev/database-toolbox/toolbox/toolbox:latest",
            "--prebuilt",
            "dataplex",
            "--stdio"
          ],
          "env": {
            "DATAPLEX_PROJECT": "your-project-id"
          }
        }
      }
    }
    ```

#### Binary Configuration

1.  **Download the Toolbox binary**:
    Download the latest binary for your operating system and architecture from the storage bucket:
    `https://storage.googleapis.com/genai-toolbox/v0.20.0/<os>/<arch>/toolbox`
    *   Replace `<os>` with `linux`, `darwin` (macOS), or `windows`.
    *   Replace `<arch>` with `amd64` (Intel) or `arm64` (Apple Silicon).

2.  **Make it executable**:
    ```bash
    chmod +x toolbox
    ```

3.  **Configure your client**:
    Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI):

    ```json
    {
      "mcpServers": {
        "dataplex": {
          "command": "./path/to/toolbox",
          "args": ["--prebuilt", "dataplex", "--stdio"],
          "env": {
            "DATAPLEX_PROJECT": "your-project-id"
          }
        }
      }
    }
    ```

## Usage

Once configured, the MCP server will automatically provide Dataplex capabilities to your AI assistant. You can:

*   "Search for entries related to 'sales' in Dataplex."
*   "Look up details for the entry 'projects/my-project/locations/us-central1/entryGroups/my-group/entries/my-entry'."

## Server Capabilities

The Dataplex MCP server provides the following tools:

| Tool Name | Description |
| :--- | :--- |
| `search_entries` | Search for entries in Dataplex Catalog. |
| `lookup_entry` | Retrieve a specific entry from Dataplex Catalog. |
| `search_aspect_types` | Find aspect types relevant to the query. |

## Documentation

For more information, visit the [Dataplex documentation](https://cloud.google.com/dataplex/docs).
