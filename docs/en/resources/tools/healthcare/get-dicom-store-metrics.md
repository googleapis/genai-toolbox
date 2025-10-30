---
title: "get-dicom-store-metrics"
linkTitle: "get-dicom-store-metrics"
type: docs
weight: 1
description: >
  A "get-dicom-store-metrics" tool retrieves metrics for a DICOM store.
aliases:
- /resources/tools/healthcare-get-dicom-store-metrics
---

## About

A `get-dicom-store-metrics` tool retrieves metrics for a DICOM store. It's
compatible with the following sources:

- [healthcare](../../sources/healthcare.md)

`get-dicom-store-metrics` returns the metrics of a DICOM store.

## Example

```yaml
tools:
  get_dicom_store_metrics:
    kind: get-dicom-store-metrics
    source: my-healthcare-source
    description: Use this tool to get metrics for a DICOM store.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                    |
|-------------|:------------------------------------------:|:------------:|----------------------------------------------------|
| kind        |                   string                   |     true     | Must be "get-dicom-store-metrics".                 |
| source      |                   string                   |     true     | Name of the healthcare source.                     |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM. |

### Parameters

| **field** |  **type**  | **required** | **description**                        |
|-----------|:----------:|:------------:|----------------------------------------|
| storeID   |   string   |     true*    | The DICOM store ID to get metrics for. |

*If the `allowedDICOMStores` in the source has length 1, then the `storeID` parameter is not needed.
