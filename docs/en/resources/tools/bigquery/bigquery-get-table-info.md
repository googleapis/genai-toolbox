---
title: "bigquery-get-table-info"
type: docs
weight: 1
description: > 
  A "bigquery-get-table-info" tool retrieves metadata for a BigQuery table.
---

## About

A `bigquery-get-table-info` tool retrieves metadata for a BigQuery table.
It's compatible with the following sources:

- [bigquery](../sources/bigquery.md)

bigquery-get-table-info takes dataset and table parameters to specify
the target table.

## Example

```yaml
tools:
  bigquery_get_table_info:
    kind: bigquery-get-table-info
    source: my-bigquery-source
    description: Use this tool to get table metadata.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "bigquery-get-table-info".                                                               |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
