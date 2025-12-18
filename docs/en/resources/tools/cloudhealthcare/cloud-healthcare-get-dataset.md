---
title: "cloud-healthcare-get-dataset"
type: docs
weight: 1
description: >
  A "cloud-healthcare-get-dataset" tool retrieves metadata for the Healthcare dataset in the source.
aliases:
- /resources/tools/cloud-healthcare-get-dataset
---

## About

A `cloud-healthcare-get-dataset` tool retrieves metadata for a Healthcare dataset.
It's compatible with the following sources:

- [cloud-healthcare](../../sources/cloud-healthcare.md)

`cloud-healthcare-get-dataset` returns the metadata of the healthcare dataset
configured in the source. It takes no extra parameters.

## Example

```yaml
tools:
  get_dataset:
    kind: cloud-healthcare-get-dataset
    source: my-healthcare-source
    description: Use this tool to get healthcare dataset metadata.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                    |
|-------------|:------------------------------------------:|:------------:|----------------------------------------------------|
| kind        |                   string                   |     true     | Must be "cloud-healthcare-get-dataset".            |
| source      |                   string                   |     true     | Name of the healthcare source.                     |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM. |
