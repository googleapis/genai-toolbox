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

`bigquery-get-dataset-info` retrieves metadata for a specific dataset. Its
behavior changes based on the source configuration:

- **Without `datasets` restriction:** The tool retrieves metadata for any dataset
  specified by the `dataset` and optional `project` parameters.
- **With `datasets` restriction:** Before retrieving metadata, the tool verifies
  that the requested dataset is in the allowed list. If it is not, the request
  is denied.

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
