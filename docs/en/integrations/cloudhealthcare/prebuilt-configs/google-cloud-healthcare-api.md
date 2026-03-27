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
