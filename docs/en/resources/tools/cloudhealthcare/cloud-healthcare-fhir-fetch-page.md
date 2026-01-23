---
title: "cloud-healthcare-fhir-fetch-page"
type: docs
weight: 1
description: >
  A "cloud-healthcare-fhir-fetch-page" tool fetches a page of FHIR resources from a given URL.
aliases:
- /resources/tools/cloud-healthcare-fhir-fetch-page
---

## About

A `cloud-healthcare-fhir-fetch-page` tool fetches a page of FHIR resources from
a given URL. It's compatible with the following sources:

- [cloud-healthcare](../../sources/cloud-healthcare.md)

`cloud-healthcare-fhir-fetch-page` can be used for pagination when a previous
tool call (like `cloud-healthcare-fhir-patient-search` or
`cloud-healthcare-fhir-patient-everything`) returns a 'next' link in the
response bundle.

## Example

```yaml
tools:
  get_fhir_store:
    kind: cloud-healthcare-fhir-fetch-page
    source: my-healthcare-source
    description: Use this tool to fetch a page of FHIR resources from a FHIR Bundle's entry.link.url
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "cloud-healthcare-fhir-fetch-page".        |
| source      |  string  |     true     | Name of the healthcare source.                     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |

### Parameters

| **field** | **type** | **required** | **description**                                                                                                                                                                               |
|-----------|:--------:|:------------:|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| pageURL   |  string  |     true     | The full URL of the FHIR page to fetch. This would usually be the value of `Bundle.entry.link.url` field within the response returned from FHIR search or FHIR patient everything operations. |
