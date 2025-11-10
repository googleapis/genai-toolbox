---
title: "cloud-healthcare-fhir-patient-everything"
type: docs
weight: 1
description: >
  A "cloud-healthcare-fhir-patient-everything" tool retrieves all information for a given patient.
aliases:
- /resources/tools/cloud-healthcare-fhir-patient-everything
---

## About

A `cloud-healthcare-fhir-patient-everything` tool retrieves resources related to a given patient
from a FHIR store. It's compatible with the following sources:

- [cloud-healthcare](../../sources/cloud-healthcare.md)

`cloud-healthcare-fhir-patient-everything` returns all the information available for a given
patient ID. It can be configured to only return certain resource types, or only
resources that have been updated after a given time.

## Example

```yaml
tools:
  fhir_patient_everything:
    kind: cloud-healthcare-fhir-patient-everything
    source: my-healthcare-source
    description: Use this tool to retrieve all the information about a given patient.
```

## Reference

| **field**   | **type** | **required** | **description**                                     |
|-------------|:--------:|:------------:|-----------------------------------------------------|
| kind        |  string  |     true     | Must be "cloud-healthcare-fhir-patient-everything". |
| source      |  string  |     true     | Name of the healthcare source.                      |
| description |  string  |     true     | Description of the tool that is passed to the LLM.  |

### Parameters

| **field**           | **type** | **required** | **description**                                                                                                                                                                                                                                                 |
|---------------------|:--------:|:------------:|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| patientID           |  string  |     true     | The ID of the patient FHIR resource for which the information is required.                                                                                                                                                                                      |
| resourceTypesFilter |  string  |    false     | String of comma-delimited FHIR resource types. If provided, only resources of the specified resource type(s) are returned.                                                                                                                                      |
| sinceFilter         |  string  |    false     | If provided, only resources updated after this time are returned. The time uses the format YYYY-MM-DDThh:mm:ss.sss+zz:zz. The time must be specified to the second and include a time zone. For example, 2015-02-07T13:28:17.239+02:00 or 2017-01-01T00:00:00Z. |
| storeID             |  string  |    true*     | The FHIR store ID to search in.                                                                                                                                                                                                                                 |

*If the `allowedFHIRStores` in the source has length 1, then the `storeID` parameter is not needed.
