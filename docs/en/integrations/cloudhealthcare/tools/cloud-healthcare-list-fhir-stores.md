---
title: "cloud-healthcare-list-fhir-stores"
type: docs
weight: 1
description: >
  A "cloud-healthcare-list-fhir-stores" lists the available FHIR stores in the healthcare dataset.
---

## About

A `cloud-healthcare-list-fhir-stores` lists the available FHIR stores in the
healthcare dataset.

`cloud-healthcare-list-fhir-stores` returns the details of the available FHIR
stores in the dataset of the healthcare source. It takes no extra parameters.


## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: list_fhir_stores
type: cloud-healthcare-list-fhir-stores
source: my-healthcare-source
description: Use this tool to list FHIR stores in the healthcare dataset.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "cloud-healthcare-list-fhir-stores".       |
| source      |  string  |     true     | Name of the healthcare source.                     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
