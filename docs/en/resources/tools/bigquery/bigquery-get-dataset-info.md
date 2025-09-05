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

`bigquery-get-dataset-info` takes a `dataset` parameter to specify the dataset
on the given source. It also optionally accepts a `project` parameter to
define the Google Cloud project ID. If the `project` parameter is not provided,
the tool defaults to using the project defined in the source configuration.

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
| type        |                   string                   |     true     | Must be "bigquery-get-dataset-info".                                                             |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
