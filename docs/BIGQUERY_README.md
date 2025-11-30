# BigQuery MCP Server

The BigQuery Model Context Protocol (MCP) Server enables AI-powered development tools to seamlessly connect, interact, and generate data insights with your BigQuery datasets and data using natural language commands.

## Features

An editor configured to use the BigQuery MCP server can use its AI capabilities to help you:

- **Natural Language to Data Analytics:** Easily find required BigQuery tables and ask analytical questions in plain English.
- **Seamless Workflow:** Stay within your CLI, eliminating the need to constantly switch to the GCP console for generating analytical insights.
- **Run Advanced Analytics:** Generate forecasts and perform contribution analysis using built-in advanced tools.

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

*   A Google Cloud project with the **BigQuery API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   BigQuery User (`roles/bigquery.user`)

## Install & Configuration

1. In the Antigravity MCP Store, click the "Install" button.

2. Add the required inputs in the configuration pop-up, then click "Save". You can update this configuration at any time in the "Configure" tab.

You'll now be able to see all enabled tools in the "Tools" tab.


### Usage 

Once configured, the MCP server will automatically provide BigQuery capabilities to your AI assistant. You can:


*   **Find Data:**

    *   "Find tables related to PyPi downloads"
    *   "Find tables related to Google analytics data in the dataset bigquery-public-data"

*   **Generate Analytics and Insights:**

    *   "Using bigquery-public-data.pypi.file_downloads show me the top 10 downloaded pypi packages this month."
    *   "Using bigquery-public-data.pypi.file_downloads can you forecast downloads for the last four months of 2025 for package urllib3?"

## Server Capabilities

The BigQuery MCP server provides the following tools:

| Tool Name              | Description                                                     |
|:-----------------------|:----------------------------------------------------------------|
| `execute_sql`          | Executes a SQL query.                                           |
| `forecast`             | Forecast time series data.                                      |
| `get_dataset_info`     | Get dataset metadata.                                           |
| `get_table_info`       | Get table metadata.                                             |
| `list_dataset_ids`     | Lists dataset IDs in the database.                              |
| `list_table_ids`       | Lists table IDs in the database.                                |
| `analyze_contribution` | Perform contribution analysis, also called key driver analysis. |
| `search_catalog`       | Search for tables based on the provided query.                  |

## Custom MCP Server Configuration

The BigQuery MCP server is configured using environment variables.

```bash
export BIGQUERY_PROJECT="<your-gcp-project-id>"
export BIGQUERY_LOCATION="<your-dataset-location>"  # Optional
export BIGQUERY_USE_CLIENT_OAUTH="true"  # Optional
```

Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI, `mcp_config.json` for Antigravity):

```json
{
  "mcpServers": {
    "bigquery": {
      "command": "toolbox",
      "args": ["--prebuilt", "bigquery", "--stdio"]
    }
  }
}
```

## Documentation

For more information, visit the [BigQuery documentation](https://cloud.google.com/bigquery/docs).
