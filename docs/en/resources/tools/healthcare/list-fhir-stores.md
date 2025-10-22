---
title: "list-fhir-stores"
linkTitle: "list-fhir-stores"
type: docs
weight: 1
description: >
  A "list-fhir-stores" lists the available FHIR stores in the healthcare dataset.
aliases:
- /resources/tools/healthcare-list-fhir-stores
---

## About

A `list-fhir-stores` lists the available FHIR stores in the healthcare dataset.
It's compatible with the following sources:

- [healthcare](../../sources/healthcare.md)

`list-fhir-stores` returns the details of the available FHIR stores in the
dataset of the healthcare source. It takes no extra parameters.

## Example

```yaml
tools:
  list_fhir_stores:
    kind: list-fhir-stores
    source: my-healthcare-source
    description: Use this tool to list FHIR stores in the healthcare dataset.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                    |
|-------------|:------------------------------------------:|:------------:|----------------------------------------------------|
| kind        |                   string                   |     true     | Must be "list-fhir-stores".                        |
| source      |                   string                   |     true     | Name of the healthcare source.                     |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM. |
