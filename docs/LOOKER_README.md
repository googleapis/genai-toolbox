# Looker MCP Server

The Looker Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Looker instance. It supports exploring models, running queries, managing dashboards, and more.

## Features

An editor configured to use the Looker MCP server can use its AI capabilities to help you:

- **Explore Models** - Get models, explores, dimensions, measures, filters, and parameters
- **Run Queries** - Execute Looker queries, generate SQL, and create query URLs
- **Manage Dashboards** - Create, run, and modify dashboards
- **Manage Looks** - Search for and run saved looks
- **Health Checks** - Analyze instance health and performance
- **Developer Tools** - Manage project files and toggle dev mode

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

*   Access to a Looker instance.
*   API Credentials (`Client ID` and `Client Secret`) or OAuth configuration.

### Configuration

1. **Access the Store**: Open the MCP Store panel within the "..." dropdown at the top of the editor's side panel.
2. **Browse and Install**: Search for "Looker", and click "Install".
3. **Configuration**: The following configuration is needed for the server:
   * Looker Base URL: The URL of your Looker instance.
   * Looker Client ID: The client ID for the Looker API.
   * Looker Client Secret: The client secret for the Looker API.
   * Looker Verify SSL: Whether to verify SSL certificates.
   * Looker Use Client OAuth: Whether to use OAuth for authentication.
   * Looker Show Hidden Models: Whether to show hidden models.
   * Looker Show Hidden Explores: Whether to show hidden explores.
   * Looker Show Hidden Fields: Whether to show hidden fields.

## Usage

Once configured, the MCP server will automatically provide Looker capabilities to your AI assistant. You can:

*   "Find explores in the 'ecommerce' model."
*   "Run a query to show total sales by month."
*   "Create a new dashboard named 'Sales Overview'."

## Server Capabilities

The Looker MCP server provides a wide range of tools. Here are some of the key capabilities:

| Tool Name               | Description                                               |
|:------------------------|:----------------------------------------------------------|
| `get_models`            | Retrieves the list of LookML models.                      |
| `get_explores`          | Retrieves the list of explores defined in a LookML model. |
| `query`                 | Run a query against the LookML model.                     |
| `query_sql`             | Generate the SQL that Looker would run.                   |
| `run_look`              | Runs a saved look.                                        |
| `run_dashboard`         | Runs all tiles in a dashboard.                            |
| `make_dashboard`        | Creates a new dashboard.                                  |
| `add_dashboard_element` | Adds a tile to a dashboard.                               |
| `health_pulse`          | Checks the status of the Looker instance.                 |
| `dev_mode`              | Toggles development mode.                                 |
| `get_projects`          | Lists LookML projects.                                    |

*(See the full list of tools in the extension)*

## Documentation

For more information, visit the [Looker documentation](https://cloud.google.com/looker/docs).
