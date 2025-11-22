# BigQuery MCP Server

The BigQuery Model Context Protocol (MCP) Server enables AI-powered development tools to seamlessly connect, interact, and generate data insights with your BigQuery datasets and data using natural language commands.

## Features

An editor configured to use the BigQuery MCP server can use its AI capabilities to help you:

- **Natural Language to Data Analytics:** Easily find required BigQuery tables and ask analytical questions in plain English.
- **Seamless Workflow:** Stay within your CLI, eliminating the need to constantly switch to the GCP console for generating analytical insights.
- **Run Advanced Analytics:** Generate forecasts and perform contribution analysis using built-in advanced tools.

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

*   A Google Cloud project with the **BigQuery API** enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.
*   IAM Permissions:
    *   BigQuery User (`roles/bigquery.user`)

### Configuration

1. **Access the Store**: Open the MCP Store panel within the "..." dropdown at the top of the editor's side panel.
2. **Browse and Install**: Search for "BigQuery", and click "Install".
3. **Configuration**: The following configuration is needed for the server:
   * BigQuery Project ID: The GCP project ID.
   * BigQuery Location: (Optional) The location of your BigQuery dataset (e.g. "US", "EU").

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

## Documentation

For more information, visit the [BigQuery documentation](https://cloud.google.com/bigquery/docs).
