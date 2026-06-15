---
title: "Knowledge Catalog (formerly known as Dataplex)"
type: docs
description: "Details of the Knowledge Catalog prebuilt configuration."
aliases:
  - /integrations/dataplex/prebuilt-configs/dataplex/
---

## Knowledge Catalog

*   `--prebuilt` value: `dataplex`
*   **Environment Variables:**
    *   `DATAPLEX_PROJECT`: The GCP project ID.
*   **Permissions:**
    *   **Dataplex Reader** (`roles/dataplex.viewer`) to search and look up
        entries.
    *   **Dataplex Editor** (`roles/dataplex.editor`) to modify entries.
*   **Tools:**
    *   `search_entries`: Searches for entries in Knowledge Catalog.
    *   `lookup_entry`: Retrieves a specific entry from Knowledge Catalog.
    *   `search_aspect_types`: Finds aspect types relevant to the
        query.
    *   `lookup_context`: Retrieves rich metadata regarding one or more data assets along with their relationships.

*   **Toolsets:**
    *   `discovery`: Tools for metadata discovery, data catalog search, and lookups.
        *   **Tools:** `search_entries`, `lookup_entry`, `search_aspect_types`, `lookup_context`, `search_dq_scans`
