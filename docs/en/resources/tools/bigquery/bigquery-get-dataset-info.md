---
title: "bigquery-get-dataset-info"
type: docs
weight: 1
description: >
  A "bigquery-get-dataset-info" tool retrieves metadata for a BigQuery dataset.
aliases:
- /resources/tools/bigquery-get-dataset-info
---

## About

A `bigquery-get-dataset-info` tool retrieves metadata for a BigQuery dataset.
It's compatible with the following sources:

- [bigquery](../../sources/bigquery.md)

`bigquery-get-dataset-info` accepts the following parameters:
- **`dataset`** (required): Specifies the dataset for which to retrieve metadata.
- **`project`** (optional): Defines the Google Cloud project ID. If not provided,
  the tool defaults to the project from the source configuration.

The tool's behavior regarding these parameters is influenced by the
`allowedDatasets` restriction on the `bigquery` source:
- **Without `allowedDatasets` restriction:** The tool can retrieve metadata for
  any dataset specified by the `dataset` and `project` parameters.
- **With `allowedDatasets` restriction:** Before retrieving metadata, the tool
  verifies that the requested dataset is in the allowed list. If it is not, the
  request is denied. If only one dataset is specified in the `allowedDatasets`
  list, it will be used as the default value for the `dataset` parameter.

## Example

```yaml
tools:
  bigquery_get_dataset_info:
    kind: bigquery-get-dataset-info
    source: my-bigquery-source
    description: Use this tool to get dataset metadata.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "bigquery-get-dataset-info".                                                             |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
