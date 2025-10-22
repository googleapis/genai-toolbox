---
title: "get-fhir-store"
linkTitle: "get-fhir-store"
type: docs
weight: 1
description: >
  A "get-fhir-store" tool retrieves information about a FHIR store.
aliases:
- /resources/tools/healthcare-get-fhir-store
---

## About

A `get-fhir-store` tool retrieves information about a FHIR store. It's
compatible with the following sources:

- [healthcare](../../sources/healthcare.md)

`get-fhir-store` returns the details of a FHIR store.

## Example

```yaml
tools:
  get_fhir_store:
    kind: get-fhir-store
    source: my-healthcare-source
    description: Use this tool to get information about a FHIR store.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                    |
|-------------|:------------------------------------------:|:------------:|----------------------------------------------------|
| kind        |                   string                   |     true     | Must be "get-fhir-store".                          |
| source      |                   string                   |     true     | Name of the healthcare source.                     |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM. |

### Parameters

| **field** |  **type**  | **required** | **description**                       |
|-----------|:----------:|:------------:|---------------------------------------|
| storeID   |   string   |     true*    | The FHIR store ID to get details for. |

*If the `allowedFHIRStores` in the source has length 1, then the `storeID` parameter is not needed.
