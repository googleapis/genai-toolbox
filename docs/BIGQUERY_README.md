# BigQuery MCP Server

The BigQuery Model Context Protocol (MCP) Server enables AI-powered development tools to seamlessly connect, interact, and generate data insights with your BigQuery datasets and data using natural language commands.

## Features

An editor configured to use the BigQuery MCP server can use its AI capabilities to help you:

- **Natural Language to Data Analytics:** Easily find required BigQuery tables and ask analytical questions in plain English.
- **Seamless Workflow:** Stay within your CLI, eliminating the need to constantly switch to the GCP console for generating analytical insights.
- **Run Advanced Analytics:** Generate forecasts and perform contribution analysis using built-in advanced tools.

## Installation and Setup

### Prerequisites

*   A Google Cloud project with the **BigQuery API** enabled.
*   Ensure the `GOOGLE_APPLICATION_CREDENTIALS` environment variable is set with [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud).
*   IAM Permissions:
    *   BigQuery User (`roles/bigquery.user`)

### Configuration

The BigQuery MCP server is configured using environment variables.

```bash
export BIGQUERY_PROJECT="<your-gcp-project-id>"
export BIGQUERY_LOCATION="<your-dataset-location>"  # Optional
export BIGQUERY_USE_CLIENT_OAUTH="true"  # Optional
```

#### Docker Configuration

1.  **Install [Docker](https://docs.docker.com/install/)**.

2.  **Configure your client**:
    Add the following configuration to your MCP client (e.g., `settings.json` for Gemini CLI):

    ```json
    {
      "mcpServers": {
        "bigquery": {
          "command": "docker",
          "args": [
            "run",
            "-i",
            "--rm",
            "-e",
            "BIGQUERY_PROJECT",
            "-e",
            "BIGQUERY_LOCATION",
            "-e",
            "GOOGLE_APPLICATION_CREDENTIALS=/tmp/keys/adc.json",
            "-v",
            "${GOOGLE_APPLICATION_CREDENTIAL}:/tmp/keys/adc.json:ro",
            "us-central1-docker.pkg.dev/database-toolbox/toolbox/toolbox:latest",
            "--prebuilt",
            "bigquery",
            "--stdio",
            ],
          "env": {
            "BIGQUERY_PROJECT": "<BIGQUERY_PROJECT>",
            "BIGQUERY_LOCATION": "<BIGQUERY_LOCATION>",
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
        "bigquery": {
          "command": "./path/to/toolbox",
          "args": ["--prebuilt", "bigquery", "--stdio"],
        }
      }
    }
    ```

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

| Tool Name | Description |
| :--- | :--- |
| `execute_sql` | Executes a SQL query. |
| `forecast` | Forecast time series data. |
| `get_dataset_info` | Get dataset metadata. |
| `get_table_info` | Get table metadata. |
| `list_dataset_ids` | Lists dataset IDs in the database. |
| `list_table_ids` | Lists table IDs in the database. |
| `analyze_contribution` | Perform contribution analysis, also called key driver analysis. |
| `search_catalog` | Search for tables based on the provided query. |

## Documentation

For more information, visit the [BigQuery documentation](https://cloud.google.com/bigquery/docs).
