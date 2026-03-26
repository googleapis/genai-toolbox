---
title: "Dataplex"
type: docs
description: "Details of the Dataplex prebuilt configuration."
---

## Dataplex

*   `--prebuilt` value: `dataplex`
*   **Environment Variables:**
    *   `DATAPLEX_PROJECT`: The GCP project ID.
*   **Permissions:**
    *   **Dataplex Reader** (`roles/dataplex.viewer`) to search and look up
        entries.
    *   **Dataplex Editor** (`roles/dataplex.editor`) to modify entries.
*   **Tools:**
    *   `search_entries`: Searches for entries in Dataplex Catalog.
    *   `lookup_entry`: Retrieves a specific entry from Dataplex
        Catalog.
    *   `search_aspect_types`: Finds aspect types relevant to the
        query.
