# Dataplex MCP Server

The Dataplex Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Google Cloud Dataplex Catalog. It supports searching and looking up entries and aspect types.

## Features

An editor configured to use the Dataplex MCP server can use its AI capabilities to help you:

- **Search Catalog** - Search for entries in Dataplex Catalog
- **Explore Metadata** - Lookup specific entries and search aspect types

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

*   A Google Cloud project with the **Dataplex API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   Dataplex Viewer (`roles/dataplex.viewer`) or equivalent permissions to read catalog entries.

### Configuration

1. **Access the Store**: Open the MCP Store panel within the "..." dropdown at the top of the editor's side panel.
2. **Browse and Install**: Search for "Dataplex", and click "Install".
3. **Configuration**: The following configuration is needed for the server:
   * Dataplex Project ID: The GCP project ID.

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

## Documentation

For more information, visit the [Dataplex documentation](https://cloud.google.com/dataplex/docs).
