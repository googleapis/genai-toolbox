# Dataplex MCP Server

The Dataplex Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud Dataplex Catalog. It supports searching and looking up entries and aspect types.

## Features

An editor configured to use the Dataplex MCP server can use its AI capabilities to help you:

- **Search Catalog** - Search for entries in Dataplex Catalog
- **Explore Metadata** - Lookup specific entries and search aspect types

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

*   A Google Cloud project with the **Dataplex API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   Dataplex Viewer (`roles/dataplex.viewer`) or equivalent permissions to read catalog entries.

## Install & Configuration

1. In the Antigravity MCP Store, click the "Install" button.

2. Add the required inputs in the configuration pop-up, then click "Save". You can update this configuration at any time in the "Configure" tab.

You'll now be able to see all enabled tools in the "Tools" tab.

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
      "command": "toolbox",
      "args": ["--prebuilt", "dataplex", "--stdio"],
      "env": {
        "DATAPLEX_PROJECT": "your-project-id"
      }
    }
  }
}
```

## Documentation

For more information, visit the [Dataplex documentation](https://cloud.google.com/dataplex/docs).
