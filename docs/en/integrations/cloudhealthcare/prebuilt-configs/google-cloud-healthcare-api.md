---
title: "Google Cloud Healthcare API"
type: docs
description: "Details of the Google Cloud Healthcare API prebuilt configuration."
---

## Google Cloud Healthcare API
*   `--prebuilt` value: `cloud-healthcare`
*   **Environment Variables:**
    *   `CLOUD_HEALTHCARE_PROJECT`: The GCP project ID.
    *   `CLOUD_HEALTHCARE_REGION`: The Cloud Healthcare API dataset region.
    *   `CLOUD_HEALTHCARE_DATASET`: The Cloud Healthcare API dataset ID.
    *   `CLOUD_HEALTHCARE_USE_CLIENT_OAUTH`: (Optional) If `true`, forwards the client's
        OAuth access token for authentication. Defaults to `false`.
*   **Permissions:**
    *   **Healthcare FHIR Resource Reader** (`roles/healthcare.fhirResourceReader`) to read an
        search FHIR resources.
    *   **Healthcare DICOM Viewer** (`roles/healthcare.dicomViewer`) to retrieve DICOM images from a
        DICOM store.
*   **Tools:**
    *   `get_dataset`: Gets information about a Cloud Healthcare API dataset.
    *   `list_dicom_stores`: Lists DICOM stores in a Cloud Healthcare API dataset.
    *   `list_fhir_stores`: Lists FHIR stores in a Cloud Healthcare API dataset.
    *   `get_fhir_store`: Gets information about a FHIR store.
    *   `get_fhir_store_metrics`: Gets metrics for a FHIR store.
    *   `get_fhir_resource`: Gets a FHIR resource from a FHIR store.
    *   `fhir_patient_search`: Searches for patient resource(s) based on a set of criteria.
    *   `fhir_patient_everything`: Retrieves resources related to a given patient.
    *   `fhir_fetch_page`: Fetches a page of FHIR resources.
    *   `get_dicom_store`: Gets information about a DICOM store.
    *   `get_dicom_store_metrics`: Gets metrics for a DICOM store.
    *   `search_dicom_studies`: Searches for DICOM studies.
    *   `search_dicom_series`: Searches for DICOM series.
    *   `search_dicom_instances`: Searches for DICOM instances.
    *   `retrieve_rendered_dicom_instance`: Retrieves a rendered DICOM instance.

*   **Toolsets:**
    *   `cloud_healthcare_dataset_tools`: Tools for managing datasets and listing stores in the Google Cloud Healthcare API.
        *   **Tools:** `get_dataset`, `list_dicom_stores`, `list_fhir_stores`
    *   `cloud_healthcare_fhir_tools`: Tools for interacting with FHIR stores and resources in the Cloud Healthcare API.
        *   **Tools:** `get_fhir_store`, `get_fhir_store_metrics`, `get_fhir_resource`, `fhir_patient_search`, `fhir_patient_everything`, `fhir_fetch_page`
    *   `cloud_healthcare_dicom_tools`: Tools for managing DICOM stores and retrieving DICOM instances in the Cloud Healthcare API.
        *   **Tools:** `get_dicom_store`, `get_dicom_store_metrics`, `search_dicom_studies`, `search_dicom_series`, `search_dicom_instances`, `retrieve_rendered_dicom_instance`
