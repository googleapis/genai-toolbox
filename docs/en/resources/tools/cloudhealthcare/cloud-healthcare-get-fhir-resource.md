---
title: "cloud-healthcare-get-fhir-resource"
linkTitle: "cloud-healthcare-get-fhir-resource"
type: docs
weight: 1
description: >
  A "cloud-healthcare-get-fhir-resource" tool retrieves a specific FHIR resource.
aliases:
- /resources/tools/cloud-healthcare-get-fhir-resource
---

## About

A `cloud-healthcare-get-fhir-resource` tool retrieves a specific FHIR resource
from a FHIR store.
It's compatible with the following sources:

- [cloud-healthcare](../../sources/cloud-healthcare.md)

`cloud-healthcare-get-fhir-resource` returns a single FHIR resource, identified
by its type and ID.

## Example

```yaml
tools:
  get_fhir_resource:
    kind: cloud-healthcare-get-fhir-resource
    source: my-healthcare-source
    description: Use this tool to retrieve a specific FHIR resource.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "cloud-healthcare-get-fhir-resource".      |
| source      |  string  |     true     | Name of the healthcare source.                     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |

### Parameters

| **field**    | **type** | **required** | **description**                                                  |
|--------------|:--------:|:------------:|------------------------------------------------------------------|
| resourceType |  string  |     true     | The FHIR resource type to retrieve (e.g., Patient, Observation). |
| resourceID   |  string  |     true     | The ID of the FHIR resource to retrieve.                         |
| storeID      |  string  |    true*     | The FHIR store ID to retrieve the resource from.                 |

*If the `allowedFHIRStores` in the source has length 1, then the `storeID`
parameter is not needed.
