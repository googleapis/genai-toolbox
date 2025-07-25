---
title: "bigquery-list-table-ids"
type: docs
weight: 1
description: > 
  A "bigquery-list-table-ids" tool returns table IDs in a given BigQuery dataset.
aliases:
- /resources/tools/bigquery-list-table-ids
---

## About

A `bigquery-list-table-ids` tool returns table IDs in a given BigQuery dataset.
It's compatible with the following sources:

- [bigquery](../sources/bigquery.md)

`bigquery-get-dataset-info` takes a required `dataset` parameter to specify the dataset
from which to list table IDs. It also optionally accepts a `project` parameter to
define the Google Cloud project ID. If the `project` parameter is not provided, the
tool defaults to using the project defined in the source configuration.

## Example

```yaml
tools:
  bigquery_list_table_ids:
    kind: bigquery-list-table-ids
    source: my-bigquery-source
    description: Use this tool to get table metadata.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "bigquery-list-table-ids".                                                               |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
