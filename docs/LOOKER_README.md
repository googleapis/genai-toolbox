# Looker MCP Server

The Looker Model Context Protocol (MCP) Server gives AI-powered development tools the ability to work with your Looker instance. It supports exploring models, running queries, managing dashboards, and more.

## Features

An editor configured to use the Looker MCP server can use its AI capabilities to help you:

- **Explore Models** - Get models, explores, dimensions, measures, filters, and parameters
- **Run Queries** - Execute Looker queries, generate SQL, and create query URLs
- **Manage Dashboards** - Create, run, and modify dashboards
- **Manage Looks** - Search for and run saved looks
- **Health Checks** - Analyze instance health and performance

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

*   Access to a Looker instance.
*   API Credentials (`Client ID` and `Client Secret`) or OAuth configuration.

## Install & Configuration

1. In the Antigravity MCP Store, click the "Install" button.

2. Add the required inputs for your [instance](https://docs.cloud.google.com/looker/docs/set-up-and-administer-looker) in the configuration pop-up, then click "Save". You can update this configuration at any time in the "Configure" tab.

You'll now be able to see all enabled tools in the "Tools" tab.

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

## Custom MCP Server Configuration

The MCP server is configured using environment variables.

```bash
export LOOKER_BASE_URL="<your-looker-instance-url>"  # e.g. `https://looker.example.com`. You may need to add the port, i.e. `:19999`.
export LOOKER_CLIENT_ID="<your-looker-client-id>"
export LOOKER_CLIENT_SECRET="<your-looker-client-secret>"
export LOOKER_VERIFY_SSL="true" # Optional, defaults to true
export LOOKER_SHOW_HIDDEN_MODELS="true" # Optional, defaults to true
export LOOKER_SHOW_HIDDEN_EXPLORES="true" # Optional, defaults to true
export LOOKER_SHOW_HIDDEN_FIELDS="true" # Optional, defaults to true
```

Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI, `mcp_config.json` for Antigravity):

```json
{
  "mcpServers": {
    "looker": {
      "command": "toolbox",
      "args": ["--prebuilt", "looker", "--stdio"],
      "env": {
        "LOOKER_BASE_URL": "https://your.looker.instance.com",
        "LOOKER_CLIENT_ID": "your-client-id",
        "LOOKER_CLIENT_SECRET": "your-client-secret"
      }
    }
  }
}
```

## Documentation

For more information, visit the [Looker documentation](https://cloud.google.com/looker).
