---
title: "fhir-patient-search"
linkTitle: "fhir-patient-search"
type: docs
weight: 1
description: >
  A "fhir-patient-search" tool searches for patients in a FHIR store.
aliases:
- /resources/tools/healthcare-fhir-patient-search
---

## About

A `fhir-patient-search` tool searches for patients in a FHIR store based on a
set of criteria. It's compatible with the following sources:

- [healthcare](../../sources/healthcare.md)

`fhir-patient-search` returns a list of patients that match the given criteria.

## Example

```yaml
tools:
  fhir_patient_search:
    kind: fhir-patient-search
    source: my-healthcare-source
    description: Use this tool to search for patients in the FHIR store.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                    |
|-------------|:------------------------------------------:|:------------:|----------------------------------------------------|
| kind        |                   string                   |     true     | Must be "fhir-patient-search".                     |
| source      |                   string                   |     true     | Name of the healthcare source.                     |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM. |

### Parameters

| **field**        |  **type**  | **required** | **description**                                                                                              |
|------------------|:----------:|:------------:|--------------------------------------------------------------------------------------------------------------|
| active           |   string   |    false     | Whether the patient record is active.                                                                        |
| city             |   string   |    false     | The city of the patient's address.                                                                           |
| country          |   string   |    false     | The country of the patient's address.                                                                        |
| postalcode       |   string   |    false     | The postal code of the patient's address.                                                                    |
| state            |   string   |    false     | The state of the patient's address.                                                                          |
| addressSubstring |   string   |    false     | A substring to search for in any address field.                                                              |
| birthDateRange   |   string   |    false     | A date range for the patient's birth date in the format YYYY-MM-DD/YYYY-MM-DD.                               |
| deathDateRange   |   string   |    false     | A date range for the patient's death date in the format YYYY-MM-DD/YYYY-MM-DD.                               |
| deceased         |   string   |    false     | Whether the patient is deceased.                                                                             |
| email            |   string   |    false     | The patient's email address.                                                                                 |
| gender           |   string   |    false     | The patient's gender.                                                                                        |
| addressUse       |   string   |    false     | The use of the patient's address.                                                                            |
| name             |   string   |    false     | The patient's name.                                                                                          |
| givenName        |   string   |    false     | A portion of the given name of the patient.                                                                  |
| familyName       |   string   |    false     | A portion of the family name of the patient.                                                                 |
| phone            |   string   |    false     | The patient's phone number.                                                                                  |
| language         |   string   |    false     | The patient's preferred language.                                                                            |
| identifier       |   string   |    false     | An identifier for the patient.                                                                               |
| summary          |  boolean   |    false     | Requests the server to return a subset of the resource. True by default.                                     |
| storeID          |   string   |    true*     | The FHIR store ID to search in.                                                                              |

*If the `allowedFHIRStores` in the source has length 1, then the `storeID` parameter is not needed.
