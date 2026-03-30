---
title: "Dataproc"
type: docs
description: "Details of the Dataproc prebuilt configuration."
---

## Dataproc

*   `--prebuilt` value: `dataproc`
*   **Environment Variables:**
    *   `DATAPROC_PROJECT`: The GCP project ID.
    *   `DATAPROC_REGION`: The Dataproc region.
*   **Permissions:**
    *   **Dataproc Viewer** (`roles/dataproc.viewer`) to examine clusters and jobs.
*   **Tools:**
    *   `list_clusters`: Lists Dataproc clusters.
    *   `get_cluster`: Gets a Dataproc cluster.
    *   `list_jobs`: Lists Dataproc jobs.
    *   `get_job`: Gets a Dataproc job.
