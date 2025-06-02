---
title: "BigQuery Prebuilt Tools"
type: docs
weight: 3
description: >
  How to get started with Toolbox's pre-built BigQuery tools.
---

## Introduction

This sample demonstrates the pre-built BigQuery tools available in Toolbox. These tools provide direct access to your BigQuery resources, allowing your agent to interact with datasets and tables dynamically. The available tools include:

* **execute_sql**: execute SQL statement
* **get_dataset_info**: get dataset metadata
* **get_table_info**: get table metadata
* **list_dataset_ids**: list datasets
* **list_table_ids**: list tables

***

## How to use

Toolbox simplifies authentication by using **Application Default Credentials (ADC)**. This method is ideal for local development and service account environments.

1.  **Authenticate your local environment** by running the following commands in your terminal. This allows Toolbox to use your user credentials to access Google Cloud services.
    ```bash
    gcloud auth login
    gcloud auth application-default login
    ```
2.  **Set your project ID** in your environment. Toolbox will use this project for all BigQuery operations.
    ```bash
    export BIGQUERY_PROJECT=YOUR_PROJECT_ID
    ```
3.  **Start Toolbox** and load the pre-built BigQuery tools using the following command:
    ```bash
    ./toolbox --prebuilt bigquery
    ```

***

## Sample prompts

Here are some example prompts you can use with an agent equipped with these tools:

* ""Which datasets exist in the project?""
* "Tell me more about the `noaa_lightning` dataset."
* "Which tables are in the `ml_datasets` dataset?"
* "Show me the schema for the `penguins` table."
* "Use SQL to compute the total population of penguins per island from the `penguins` table."

## Related Guides

For more information on how to integrate and run Toolbox in different environments, check out the following guides:

* **[Connecting to an Agent (LangGraph, LlamaIndex, ADK)](./local_quickstart.md)**
* **[Running with MCP (sse)](./mcp_quickstart/_index.md)**
* **[Running with MCP (Stdio)](../../how-to/connect-ide/bigquery_mcp.md)**