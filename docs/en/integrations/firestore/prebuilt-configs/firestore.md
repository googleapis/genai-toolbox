---
title: "Firestore"
type: docs
description: "Details of the Firestore prebuilt configuration."
---

## Firestore

*   `--prebuilt` value: `firestore`
*   **Environment Variables:**
    *   `FIRESTORE_PROJECT`: The GCP project ID.
    *   `FIRESTORE_DATABASE`: (Optional) The Firestore database ID. Defaults to
        "(default)".
*   **Permissions:**
    *   **Cloud Datastore User** (`roles/datastore.user`) to get documents, list
        collections, and query collections.
    *   **Firebase Rules Viewer** (`roles/firebaserules.viewer`) to get and
        validate Firestore rules.
*   **Tools:**
    *   `get_documents`: Gets multiple documents from Firestore by their paths.
    *   `add_documents`: Adds a new document to a Firestore collection.
    *   `update_document`: Updates an existing document in Firestore.
    *   `list_collections`: Lists Firestore collections for a given parent path.
    *   `delete_documents`: Deletes multiple documents from Firestore.
    *   `query_collection`: Retrieves one or more Firestore documents from a
        collection.
    *   `get_rules`: Retrieves the active Firestore security rules.
    *   `validate_rules`: Checks the provided Firestore Rules source for syntax
        and validation errors.
