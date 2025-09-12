---
title: "bigquery-forecast"
type: docs
weight: 1
description: >
  A "bigquery-forecast" tool forecasts time series data in BigQuery.
aliases:
- /resources/tools/bigquery-forecast
---

## About

A `bigquery-forecast` tool forecasts time series data in BigQuery.
It's compatible with the following sources:

- [bigquery](../../sources/bigquery.md)

`bigquery-forecast` constructs and executes a `SELECT * FROM AI.FORECAST(...)` query. Its behavior changes based on the source configuration:

- **Without `allowed_datasets` restriction:** The tool can use any table or query for the `history_data` parameter.
- **With `allowed_datasets` restriction:** The tool verifies that the `history_data` parameter only accesses tables within the allowed datasets.
  - If `history_data` is a table ID, the tool checks if the table's dataset is in the allowed list.
  - If `history_data` is a query, the tool performs a dry run to analyze the query and rejects it if it accesses any table outside the allowed list.

The tool uses the following parameters:

- **history_data** (string, required): This specifies the source of the historical time series data. It can be either a fully qualified BigQuery table ID (e.g., my-project.my_dataset.my_table) or a SQL query that returns the data.
- **timestamp_col** (string, required): The name of the column in your history_data that contains the timestamps.
- **data_col** (string, required): The name of the column in your history_data that contains the numeric values to be forecasted.
- **id_cols** (array of strings, optional): If you are forecasting multiple time series at once (e.g., sales for different products), this parameter takes an array of column names that uniquely identify each series. It defaults to an empty array if not provided.
- **horizon** (integer, optional): The number of future time steps you want to predict. It defaults to 10 if not specified.

## Example

```yaml
tools:
 forecast_tool:
    kind: bigquery-forecast
    source: my-bigquery-source
    description: Use this tool to forecast time series data in BigQuery.
```

## Sample Prompt
You can use the following sample prompts to call this tool:

- Can you forecast the history time series data in bigquery table `bqml_tutorial.google_analytic`? Use project_id `myproject`.
- What are the future `total_visits` in bigquery table `bqml_tutorial.google_analytic`?


## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "bigquery-forecast".                                                                  |
| source      |                   string                   |     true     | Name of the source the forecast tool should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
