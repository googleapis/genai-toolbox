---
title: "cloud-healthcare-list-dicom-stores"
type: docs
weight: 1
description: >
  A "cloud-healthcare-list-dicom-stores" lists the available DICOM stores in the healthcare dataset.
aliases:
- /resources/tools/cloud-healthcare-list-dicom-stores
---

## About

A `cloud-healthcare-list-dicom-stores` lists the available DICOM stores in the
healthcare dataset.
It's compatible with the following sources:

- [cloud-healthcare](../../sources/cloud-healthcare.md)

`cloud-healthcare-list-dicom-stores` returns the details of the available DICOM
stores in the dataset of the healthcare source. It takes no extra parameters.

## Example

```yaml
tools:
  list_dicom_stores:
    kind: cloud-healthcare-list-dicom-stores
    source: my-healthcare-source
    description: Use this tool to list DICOM stores in the healthcare dataset.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "cloud-healthcare-list-dicom-stores".      |
| source      |  string  |     true     | Name of the healthcare source.                     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
