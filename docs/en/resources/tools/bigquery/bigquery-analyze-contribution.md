---
title: "bigquery-analyze-contribution"
type: docs
weight: 1
description: >
  A "bigquery-analyze-contribution" tool performs contribution analysis in BigQuery.
aliases:
- /resources/tools/bigquery-analyze-contribution
---

## About

A `bigquery-analyze-contribution` tool performs contribution analysis in
BigQuery by creating a temporary `CONTRIBUTION_ANALYSIS` model and then querying
it with `ML.GET_INSIGHTS` to find top contributors for a given metric.

It's compatible with the following sources:

- [bigquery](../../sources/bigquery.md)

`bigquery-analyze-contribution` takes the following parameters:

- **input_data** (string, required): The data that contain the test and control
  data to analyze. This can be a fully qualified BigQuery table ID (e.g.,
  `my-project.my_dataset.my_table`) or a SQL query that returns the data.
- **contribution_metric** (string, required): The name of the column that
  contains the metric to analyze. This can be SUM(metric_column_name),
  SUM(numerator_metric_column_name)/SUM(denominator_metric_column_name) or
  SUM(metric_sum_column_name)/COUNT(DISTINCT categorical_column_name) depending
  the type of metric to analyze.
- **is_test_col** (string, required): The name of the column that identifies
  whether a row is in the test or control group. The column must contain boolean
  values.
- **dimension_id_cols** (array of strings, optional): An array of column names
  that uniquely identify each dimension.
- **top_k_insights_by_apriori_support** (integer, optional): The number of top
  insights to return, ranked by apriori support. Default to '30'.
- **pruning_method** (string, optional): The method to use for pruning redundant
  insights. Can be `'NO_PRUNING'` or `'PRUNE_REDUNDANT_INSIGHTS'`. Defaults to
  `'PRUNE_REDUNDANT_INSIGHTS'`.


## Example

```yaml
tools:
  contribution_analyzer:
    kind: bigquery-analyze-contribution
    source: my-bigquery-source
    description: Use this tool to run contribution analysis on a dataset in BigQuery.
```

## Sample Prompt
You can prepare a sample table following
https://cloud.google.com/bigquery/docs/get-contribution-analysis-insights.
And use the following sample prompts to call this tool:

- What drives the changes in sales in the table
  `bqml_tutorial.iowa_liquor_sales_sum_data`? Use the project id myproject.
- Analyze the contribution for the `total_sales` metric in the table
  `bqml_tutorial.iowa_liquor_sales_sum_data`. The test group is identified by
  the `is_test` column. The dimensions are `store_name`, `city`, `vendor_name`,
  `category_name` and `item_description`.

## Reference

| **field**   | **type** | **required** | **description**                                            |
|-------------|:--------:|:------------:|------------------------------------------------------------|
| kind        |  string  |     true     | Must be "bigquery-analyze-contribution".                   |
| source      |  string  |     true     | Name of the source the tool should execute on.             |
| description |  string  |     true     | Description of the tool that is passed to the LLM.         |
