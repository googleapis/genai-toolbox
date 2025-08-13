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

`bigquery-list-dataset-ids` lists available dataset IDs. Its behavior changes
based on the source configuration:

- **Without `datasets` restriction:** The tool lists all available datasets in
  the specified Google Cloud project. It accepts an optional `project`
  parameter; if not provided, it defaults to the project in the source
  configuration.
- **With `datasets` restriction:** The tool directly returns the pre-configured
  list of dataset IDs from the source. The `project` parameter is ignored.

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
