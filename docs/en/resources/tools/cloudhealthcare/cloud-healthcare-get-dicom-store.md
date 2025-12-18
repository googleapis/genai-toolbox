---
title: "cloud-healthcare-get-dicom-store"
type: docs
weight: 1
description: >
  A "cloud-healthcare-get-dicom-store" tool retrieves information about a DICOM store.
aliases:
- /resources/tools/cloud-healthcare-get-dicom-store
---

## About

A `cloud-healthcare-get-dicom-store` tool retrieves information about a DICOM store. It's
compatible with the following sources:

- [cloud-healthcare](../../sources/cloud-healthcare.md)

`cloud-healthcare-get-dicom-store` returns the details of a DICOM store.

## Example

```yaml
tools:
  get_dicom_store:
    kind: cloud-healthcare-get-dicom-store
    source: my-healthcare-source
    description: Use this tool to get information about a DICOM store.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "cloud-healthcare-get-dicom-store".        |
| source      |  string  |     true     | Name of the healthcare source.                     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |

### Parameters

| **field** | **type** | **required** | **description**                        |
|-----------|:--------:|:------------:|----------------------------------------|
| storeID   |  string  |    true*     | The DICOM store ID to get details for. |

*If the `allowedDICOMStores` in the source has length 1, then the `storeID` parameter is not needed.
