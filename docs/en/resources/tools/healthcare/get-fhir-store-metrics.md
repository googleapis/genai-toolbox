---
title: "get-fhir-store-metrics"
linkTitle: "get-fhir-store-metrics"
type: docs
weight: 1
description: >
  A "get-fhir-store-metrics" tool retrieves metrics for a FHIR store.
aliases:
- /resources/tools/healthcare-get-fhir-store-metrics
---

## About

A `get-fhir-store-metrics` tool retrieves metrics for a FHIR store. It's
compatible with the following sources:

- [healthcare](../../sources/healthcare.md)

`get-fhir-store-metrics` returns the metrics of a FHIR store.

## Example

```yaml
tools:
  get_fhir_store_metrics:
    kind: get-fhir-store-metrics
    source: my-healthcare-source
    description: Use this tool to get metrics for a FHIR store.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                    |
|-------------|:------------------------------------------:|:------------:|----------------------------------------------------|
| kind        |                   string                   |     true     | Must be "get-fhir-store-metrics".                  |
| source      |                   string                   |     true     | Name of the healthcare source.                     |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM. |

### Parameters

| **field** |  **type**  | **required** | **description**                       |
|-----------|:----------:|:------------:|---------------------------------------|
| storeID   |   string   |     true*    | The FHIR store ID to get metrics for. |

*If the `allowedFHIRStores` in the source has length 1, then the `storeID` parameter is not needed.
