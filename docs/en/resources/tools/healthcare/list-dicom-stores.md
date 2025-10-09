---
title: "list-dicom-stores"
linkTitle: "list-dicom-stores"
type: docs
weight: 1
description: >
  A "list-dicom-stores" lists the available DICOM stores in the healthcare dataset.
aliases:
- /resources/tools/healthcare-list-dicom-stores
---

## About

A `list-dicom-stores` lists the available DICOM stores in the healthcare dataset.
It's compatible with the following sources:

- [healthcare](../../sources/healthcare.md)

`list-dicom-stores` returns the details of the available DICOM stores in the
dataset of the healthcare source. It takes no extra parameters.

## Example

```yaml
tools:
  list_dicom_stores:
    kind: list-dicom-stores
    source: my-healthcare-source
    description: Use this tool to list DICOM stores in the healthcare dataset.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                    |
|-------------|:------------------------------------------:|:------------:|----------------------------------------------------|
| kind        |                   string                   |     true     | Must be "list-dicom-stores".                       |
| source      |                   string                   |     true     | Name of the healthcare source.                     |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM. |
