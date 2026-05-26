---
title: "cloud-healthcare-get-dicom-store-metrics"
type: docs
weight: 1
description: >
  A "cloud-healthcare-get-dicom-store-metrics" tool retrieves metrics for a DICOM store.
---

## About

A `cloud-healthcare-get-dicom-store-metrics` tool retrieves metrics for a DICOM
store.

`cloud-healthcare-get-dicom-store-metrics` returns the metrics of a DICOM store.


## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: get_dicom_store_metrics
type: cloud-healthcare-get-dicom-store-metrics
source: my-healthcare-source
description: Use this tool to get metrics for a DICOM store.
```

## Reference

| **field**   | **type** | **required** | **description**                                     |
|-------------|:--------:|:------------:|-----------------------------------------------------|
| type        |  string  |     true     | Must be "cloud-healthcare-get-dicom-store-metrics". |
| source      |  string  |     true     | Name of the healthcare source.                      |
| description |  string  |     true     | Description of the tool that is passed to the LLM.  |

### Parameters

| **field** | **type** | **required** | **description**                        |
|-----------|:--------:|:------------:|----------------------------------------|
| storeID   |  string  |    true*     | The DICOM store ID to get metrics for. |

*If the `allowedDICOMStores` in the source has length 1, then the `storeID`
parameter is not needed.
