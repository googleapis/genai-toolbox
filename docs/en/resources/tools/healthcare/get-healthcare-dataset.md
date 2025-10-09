---
title: "get-healthcare-dataset"
linkTitle: "get-healthcare-dataset"
type: docs
weight: 1
description: >
  A "get-healthcare-dataset" tool retrieves metadata for the Healthcare dataset in the source.
aliases:
- /resources/tools/healthcare-get-healthcare-dataset
---

## About

A `get-healthcare-dataset` tool retrieves metadata for a Healthcare dataset.
It's compatible with the following sources:

- [healthcare](../../sources/healthcare.md)

`get-healthcare-dataset` returns the metadata of the healthcare dataset
configured in the source. It takes no extra parameters.

## Example

```yaml
tools:
  get_healthcare_dataset:
    kind: get-healthcare-dataset
    source: my-healthcare-source
    description: Use this tool to get healthcare dataset metadata.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                    |
|-------------|:------------------------------------------:|:------------:|----------------------------------------------------|
| kind        |                   string                   |     true     | Must be "get-healthcare-dataset".                  |
| source      |                   string                   |     true     | Name of the healthcare source.                     |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM. |
