---
title: "cloud-healthcare-search-dicom-series"
type: docs
weight: 1
description: >
  A "cloud-healthcare-search-dicom-series" tool searches for DICOM series in a DICOM store.
aliases:
- /resources/tools/cloud-healthcare-search-dicom-series
---

## About

A `cloud-healthcare-search-dicom-series` tool searches for DICOM series in a DICOM store based on a
set of criteria. It's compatible with the following sources:

- [cloud-healthcare](../../sources/cloud-healthcare.md)

`cloud-healthcare-search-dicom-series` returns a list of DICOM series that match the given criteria.

## Example

```yaml
tools:
  search_dicom_series:
    kind: cloud-healthcare-search-dicom-series
    source: my-healthcare-source
    description: Use this tool to search for DICOM series in the DICOM store.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "cloud-healthcare-search-dicom-series".    |
| source      |  string  |     true     | Name of the healthcare source.                     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |

### Parameters

| **field**              | **type** | **required** | **description**                                                                                                                                                                                                                                                                                                                                                               |
|------------------------|:--------:|:------------:|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| StudyInstanceUID       |  string  |    false     | The UID of the DICOM study.                                                                                                                                                                                                                                                                                                                                                   |
| PatientName            |  string  |    false     | The name of the patient.                                                                                                                                                                                                                                                                                                                                                      |
| PatientID              |  string  |    false     | The ID of the patient.                                                                                                                                                                                                                                                                                                                                                        |
| AccessionNumber        |  string  |    false     | The accession number of the study.                                                                                                                                                                                                                                                                                                                                            |
| ReferringPhysicianName |  string  |    false     | The name of the referring physician.                                                                                                                                                                                                                                                                                                                                          |
| StudyDate              |  string  |    false     | The date of the study in the format `YYYYMMDD`. You can also specify a date range in the format `YYYYMMDD-YYYYMMDD`.                                                                                                                                                                                                                                                          |
| SeriesInstanceUID      |  string  |    false     | The UID of the DICOM series.                                                                                                                                                                                                                                                                                                                                                  |
| Modality               |  string  |    false     | The modality of the series.                                                                                                                                                                                                                                                                                                                                                   |
| fuzzymatching          | boolean  |    false     | Whether to enable fuzzy matching for patient names. Fuzzy matching will perform tokenization and normalization of both the value of PatientName in the query and the stored value. It will match if any search token is a prefix of any stored token. For example, if PatientName is "John^Doe", then "jo", "Do" and "John Doe" will all match. However "ohn" will not match. |
| includefield           | []string |    false     | List of attributeIDs to include in the output, such as DICOM tag IDs or keywords. Set to `["all"]` to return all available tags.                                                                                                                                                                                                                                              |
| storeID                |  string  |    true*     | The DICOM store ID to search in.                                                                                                                                                                                                                                                                                                                                              |

*If the `allowedDIComStores` in the source has length 1, then the `storeID` parameter is not needed.
