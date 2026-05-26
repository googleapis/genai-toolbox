---
title: "cloud-healthcare-get-fhir-store"
type: docs
weight: 1
description: >
  A "cloud-healthcare-get-fhir-store" tool retrieves information about a FHIR store.

---

## About

A `cloud-healthcare-get-fhir-store` tool retrieves information about a FHIR store.

`cloud-healthcare-get-fhir-store` returns the details of a FHIR store.

## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: get_fhir_store
type: cloud-healthcare-get-fhir-store
source: my-healthcare-source
description: Use this tool to get information about a FHIR store.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "cloud-healthcare-get-fhir-store".         |
| source      |  string  |     true     | Name of the healthcare source.                     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |

### Parameters

| **field** | **type** | **required** | **description**                       |
|-----------|:--------:|:------------:|---------------------------------------|
| storeID   |  string  |    true*     | The FHIR store ID to get details for. |

*If the `allowedFHIRStores` in the source has length 1, then the `storeID` parameter is not needed.
