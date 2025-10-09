---
title: "bigquery-list-dataset-ids"
type: docs
weight: 1
description: >
  A "bigquery-list-dataset-ids" tool returns all dataset IDs from the source.
aliases:
- /resources/tools/bigquery-list-dataset-ids
---

## About

A `bigquery-list-dataset-ids` tool returns all dataset IDs from the source.
It's compatible with the following sources:

- [bigquery](../../sources/bigquery.md)

`bigquery-list-dataset-ids` accepts the following parameter:
- **`project`** (optional): Defines the Google Cloud project ID. If not provided,
  the tool defaults to the project from the source configuration.

The tool's behavior regarding this parameter is influenced by the
`allowedDatasets` restriction on the `bigquery` source:
- **Without `allowedDatasets` restriction:** The tool can list datasets from any
  project specified by the `project` parameter.
- **With `allowedDatasets` restriction:** The tool directly returns the
  pre-configured list of dataset IDs from the source, and the `project`
  parameter is ignored.

## Example

```yaml
tools:
  bigquery_list_dataset_ids:
    kind: bigquery-list-dataset-ids
    source: my-bigquery-source
    description: Use this tool to get dataset metadata.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "bigquery-list-dataset-ids".                                                             |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
