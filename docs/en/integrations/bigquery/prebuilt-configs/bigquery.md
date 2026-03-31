---
title: "BigQuery"
type: docs
description: "Details of the BigQuery prebuilt configuration."
---

## BigQuery

*   `--prebuilt` value: `bigquery`
*   **Environment Variables:**
    *   `BIGQUERY_PROJECT`: The GCP project ID.
    *   `BIGQUERY_LOCATION`: (Optional) The dataset location.
    *   `BIGQUERY_USE_CLIENT_OAUTH`: (Optional) If `true`, forwards the client's
        OAuth access token for authentication. Defaults to `false`.
    *   `BIGQUERY_SCOPES`: (Optional) A comma-separated list of OAuth scopes to
        use for authentication.
    *   `BIGQUERY_IMPERSONATE_SERVICE_ACCOUNT`: (Optional) Service account email
        to impersonate when making BigQuery and Dataplex API calls. The
        authenticated principal must have `roles/iam.serviceAccountTokenCreator`
        on the target service account.
*   **Permissions:**
    *   **BigQuery User** (`roles/bigquery.user`) to execute queries and view
        metadata.
    *   **BigQuery Metadata Viewer** (`roles/bigquery.metadataViewer`) to view
        all datasets.
    *   **BigQuery Data Editor** (`roles/bigquery.dataEditor`) to create or
        modify datasets and tables.
    *   **Gemini for Google Cloud** (`roles/cloudaicompanion.user`) to use the
        conversational analytics API.
*   **Tools:**
    *   `analyze_contribution`: Use this tool to perform contribution analysis,
        also called key driver analysis.
    *   `ask_data_insights`: Use this tool to perform data analysis, get
        insights, or answer complex questions about the contents of specific
        BigQuery tables. For more information on required roles, API setup, and
        IAM configuration, see the setup and authentication section of the
        [Conversational Analytics API
        documentation](https://cloud.google.com/gemini/docs/conversational-analytics-api/overview).
    *   `execute_sql`: Executes a SQL statement.
    *   `forecast`: Use this tool to forecast time series data.
    *   `get_dataset_info`: Gets dataset metadata.
    *   `get_table_info`: Gets table metadata.
    *   `list_dataset_ids`: Lists datasets.
    *   `list_table_ids`: Lists tables.
    *   `search_catalog`: Search for entries based on the provided query.
