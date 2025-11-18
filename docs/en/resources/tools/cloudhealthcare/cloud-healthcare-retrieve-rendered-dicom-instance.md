---
title: "cloud-healthcare-retrieve-rendered-dicom-instance"
type: docs
weight: 1
description: >
  A "cloud-healthcare-retrieve-rendered-dicom-instance" tool retrieves a rendered DICOM instance from a DICOM store.
aliases:
- /resources/tools/cloud-healthcare-retrieve-rendered-dicom-instance
---

## About

A `cloud-healthcare-retrieve-rendered-dicom-instance` tool retrieves a rendered
DICOM instance from a DICOM store.
It's compatible with the following sources:

- [cloud-healthcare](../../sources/cloud-healthcare.md)

`cloud-healthcare-retrieve-rendered-dicom-instance` returns a base64 encoded
string of the image in JPEG format.

## Example

```yaml
tools:
  retrieve_rendered_dicom_instance:
    kind: cloud-healthcare-retrieve-rendered-dicom-instance
    source: my-healthcare-source
    description: Use this tool to retrieve a rendered DICOM instance from the DICOM store.
```

## Reference

| **field**   | **type** | **required** | **description**                                              |
|-------------|:--------:|:------------:|--------------------------------------------------------------|
| kind        |  string  |     true     | Must be "cloud-healthcare-retrieve-rendered-dicom-instance". |
| source      |  string  |     true     | Name of the healthcare source.                               |
| description |  string  |     true     | Description of the tool that is passed to the LLM.           |

### Parameters

| **field**         | **type** | **required** | **description**                                                                                  |
|-------------------|:--------:|:------------:|--------------------------------------------------------------------------------------------------|
| StudyInstanceUID  |  string  |     true     | The UID of the DICOM study.                                                                      |
| SeriesInstanceUID |  string  |     true     | The UID of the DICOM series.                                                                     |
| SOPInstanceUID    |  string  |     true     | The UID of the SOP instance.                                                                     |
| FrameNumber       | integer  |    false     | The frame number to retrieve (1-based). Only applicable to multi-frame instances. Defaults to 1. |
| storeID           |  string  |    true*     | The DICOM store ID to retrieve from.                                                             |

*If the `allowedDICOMStores` in the source has length 1, then the `storeID`
parameter is not needed.
