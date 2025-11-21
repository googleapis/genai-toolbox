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

  3.  **Add the binary to $PATH in `.~/bash_profile`**:
      ```bash
      export PATH=$PATH:/path/to/toolbox
      ```
    
**Note:** You may need to restart Antigravity for changes to take effect. 
Windows OS users will need to follow one of the Windows-specific methods.

*   Access to a Looker instance.
*   API Credentials (`Client ID` and `Client Secret`) or OAuth configuration.

### Configuration

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

Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI):

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

## Usage

Once configured, the MCP server will automatically provide Looker capabilities to your AI assistant. You can:

*   "Find explores in the 'ecommerce' model."
*   "Run a query to show total sales by month."
*   "Create a new dashboard named 'Sales Overview'."

## Server Capabilities

The Looker MCP server provides a wide range of tools. Here are some of the key capabilities:

| Tool Name | Description |
| :--- | :--- |
| `get_models` | Retrieves the list of LookML models. |
| `get_explores` | Retrieves the list of explores defined in a LookML model. |
| `query` | Run a query against the LookML model. |
| `query_sql` | Generate the SQL that Looker would run. |
| `run_look` | Runs a saved look. |
| `run_dashboard` | Runs all tiles in a dashboard. |
| `make_dashboard` | Creates a new dashboard. |
| `add_dashboard_element` | Adds a tile to a dashboard. |
| `health_pulse` | Checks the status of the Looker instance. |
| `dev_mode` | Toggles development mode. |
| `get_projects` | Lists LookML projects. |

*(See the full list of tools in the extension)*

## Documentation

For more information, visit the [Looker documentation](https://cloud.google.com/looker/docs).
