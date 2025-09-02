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

- [bigquery](../../sources/bigquery.md)

`bigquery-list-table-ids` accepts the following parameters:
- **`dataset`** (required): Specifies the dataset from which to list table IDs.
- **`project`** (optional): Defines the Google Cloud project ID. If not provided,
the tool defaults to the project from the source configuration.

The tool's behavior regarding these parameters is influenced by the 
`allowedDatasets` restriction on the `bigquery` source:
- **Without `allowedDatasets` restriction:** The tool can list tables from any 
dataset specified by the `dataset` and `project` parameters.
- **With `allowedDatasets` restriction:** Before listing tables, the tool verifies
that the requested dataset is in the allowed list. If it is not, the request is
denied. If only one dataset is specified in the `allowedDatasets` list, it
will be used as the default value for the `dataset` parameter.

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
