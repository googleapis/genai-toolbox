---
title: "Cloud Healthcare API"
linkTitle: "Cloud Healthcare"
type: docs
weight: 1
description: >
  The Cloud Healthcare API provides a managed solution for storing and
  accessing healthcare data in Google Cloud, providing a critical bridge 
  between existing care systems and applications hosted on Google Cloud.
---

## About

The [Cloud Healthcare API][healthcare-docs] provides a managed solution
for storing and accessing healthcare data in Google Cloud, providing a
critical bridge between existing care systems and applications hosted on
Google Cloud. It supports healthcare data standards such as HL7® FHIR®,
HL7® v2, and DICOM®. It provides a fully managed, highly scalable,
enterprise-grade development environment for building clinical and analytics
solutions securely on Google Cloud.

A dataset is a container in your Google Cloud project that holds modality-specific
healthcare data. Datasets contain other data stores, such as FHIR stores and DICOM
stores, which in turn hold their own types of healthcare data.

A single dataset can contain one or many data stores, and those stores can all
service the same modality or different modalities as application needs dictate.
Using multiple stores in the same dataset might be appropriate in various
situations.

If you are new to the Cloud Healthcare API, you can try to
[create and view datasets and stores using curl][healthcare-quickstart-curl].

[healthcare-docs]: https://cloud.google.com/healthcare/docs
[healthcare-quickstart-curl]:
    https://cloud.google.com/healthcare-api/docs/store-healthcare-data-rest

## Available Tools

- [`cloud-healthcare-get-dataset`](../tools/cloudhealthcare/cloud-healthcare-get-dataset.md)
  Retrieves a dataset’s details.

- [`cloud-healthcare-list-fhir-stores`](../tools/cloudhealthcare/cloud-healthcare-list-fhir-stores.md)
  Lists the available FHIR stores in the healthcare dataset.

- [`cloud-healthcare-list-dicom-stores`](../tools/cloudhealthcare/cloud-healthcare-list-dicom-stores.md)
  Lists the available DICOM stores in the healthcare dataset.

- [`cloud-healthcare-get-fhir-store`](../tools/cloudhealthcare/cloud-healthcare-get-fhir-store.md)
  Retrieves information about a FHIR store.

- [`cloud-healthcare-get-fhir-store-metrics`](../tools/cloudhealthcare/cloud-healthcare-get-fhir-store-metrics.md)
  Retrieves metrics for a FHIR store.

- [`cloud-healthcare-get-fhir-resource`](../tools/cloudhealthcare/cloud-healthcare-get-fhir-resource.md)
  Retrieves a specific FHIR resource from a FHIR store.

- [`cloud-healthcare-fhir-patient-search`](../tools/cloudhealthcare/cloud-healthcare-fhir-patient-search.md)
  Searches for patients in a FHIR store based on a set of criteria.

- [`cloud-healthcare-fhir-patient-everything`](../tools/cloudhealthcare/cloud-healthcare-fhir-patient-everything.md)
  Retrieves all information for a given patient.

- [`cloud-healthcare-fhir-fetch-page`](../tools/cloudhealthcare/cloud-healthcare-fhir-fetch-page.md)
  Fetches a page of FHIR resources from a given URL.

- [`cloud-healthcare-get-dicom-store`](../tools/cloudhealthcare/cloud-healthcare-get-dicom-store.md)
  Retrieves information about a DICOM store.

- [`cloud-healthcare-get-dicom-store-metrics`](../tools/cloudhealthcare/cloud-healthcare-get-dicom-store-metrics.md)
  Retrieves metrics for a DICOM store.

- [`cloud-healthcare-search-dicom-studies`](../tools/cloudhealthcare/cloud-healthcare-search-dicom-studies.md)
  Searches for DICOM studies in a DICOM store.

- [`cloud-healthcare-search-dicom-series`](../tools/cloudhealthcare/cloud-healthcare-search-dicom-series.md)
  Searches for DICOM series in a DICOM store.

- [`cloud-healthcare-search-dicom-instances`](../tools/cloudhealthcare/cloud-healthcare-search-dicom-instances.md)
  Searches for DICOM instances in a DICOM store.

- [`cloud-healthcare-retrieve-rendered-dicom-instance`](../tools/cloudhealthcare/cloud-healthcare-retrieve-rendered-dicom-instance.md)
  Retrieves a rendered DICOM instance from a DICOM store.

## Requirements

### IAM Permissions

The Cloud Healthcare API uses [Identity and Access Management
(IAM)][iam-overview] to control user and group access to Cloud Healthcare
resources like projects, datasets, and stores.

### Authentication via Application Default Credentials (ADC)

By **default**, Toolbox will use your [Application Default Credentials
(ADC)][adc] to authorize and authenticate when interacting with the
[Cloud Healthcare API][healthcare-docs].

When using this method, you need to ensure the IAM identity associated with your
ADC (such as a service account) has the correct permissions for the queries you
intend to run. Common roles include `roles/healthcare.fhirResourceReader` (which
includes permissions to read and search for FHIR resources) or
`roles/healthcare.dicomViewer` (for retrieving DICOM images).
Follow this [guide][set-adc] to set up your ADC.

### Authentication via User's OAuth Access Token

If the `useClientOAuth` parameter is set to `true`, Toolbox will instead use the
OAuth access token for authentication. This token is parsed from the
`Authorization` header passed in with the tool invocation request. This method
allows Toolbox to make queries to the [Cloud Healthcare API][healthcare-docs] on
behalf of the client or the end-user.

When using this on-behalf-of authentication, you must ensure that the
identity used has been granted the correct IAM permissions.

[iam-overview]: <https://cloud.google.com/healthcare/docs/access-control>
[adc]: <https://cloud.google.com/docs/authentication#adc>
[set-adc]: <https://cloud.google.com/docs/authentication/provide-credentials-adc>

## Example

Initialize a Cloud Healthcare API source that uses ADC:

```yaml
sources:
  my-healthcare-source:
    kind: "cloud-healthcare"
    project: "my-project-id"
    region: "us-central1"
    dataset: "my-healthcare-dataset-id"
    # allowedFhirStores: # Optional: Restricts tool access to a specific list of FHIR store IDs.
    #   - "my_fhir_store_1"
    # allowedDicomStores: # Optional: Restricts tool access to a specific list of DICOM store IDs.
    #   - "my_dicom_store_1"
    #   - "my_dicom_store_2"
```

Initialize a Cloud Healthcare API source that uses the client's access token:

```yaml
sources:
  my-healthcare-client-auth-source:
    kind: "cloud-healthcare"
    project: "my-project-id"
    region: "us-central1"
    dataset: "my-healthcare-dataset-id"
    useClientOAuth: true
    # allowedFhirStores: # Optional: Restricts tool access to a specific list of FHIR store IDs.
    #   - "my_fhir_store_1"
    # allowedDicomStores: # Optional: Restricts tool access to a specific list of DICOM store IDs.
    #   - "my_dicom_store_1"
    #   - "my_dicom_store_2"
```

## Reference

| **field**          | **type** | **required** | **description**                                                                                                                                                                                                                                                              |
|--------------------|:--------:|:------------:|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| kind               |  string  |     true     | Must be "cloud-healthcare".                                                                                                                                                                                                                                                  |
| project            |  string  |     true     | ID of the GCP project that the dataset lives in.                                                                                                                                                                                                                             |
| region             |  string  |     true     | Specifies the region (e.g., 'us', 'asia-northeast1') of the healthcare dataset. [Learn More](https://cloud.google.com/healthcare-api/docs/regions)                                                                                                                           |
| dataset            |  string  |     true     | ID of the healthcare dataset.                                                                                                                                                                                                                                                |
| allowedFhirStores  | []string |    false     | An optional list of FHIR store IDs that tools using this source are allowed to access. If provided, any tool operation attempting to access a store not in this list will be rejected. If a single store is provided, it will be treated as the default for prebuilt tools.  |
| allowedDicomStores | []string |    false     | An optional list of DICOM store IDs that tools using this source are allowed to access. If provided, any tool operation attempting to access a store not in this list will be rejected. If a single store is provided, it will be treated as the default for prebuilt tools. |
| useClientOAuth     |   bool   |    false     | If true, forwards the client's OAuth access token from the "Authorization" header to downstream queries.                                                                                                                                                                     |
