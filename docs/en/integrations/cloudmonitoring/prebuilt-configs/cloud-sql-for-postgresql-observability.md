---
title: "Cloud SQL for PostgreSQL Observability"
type: docs
description: "Details of the Cloud SQL for PostgreSQL Observability prebuilt configuration."
---

## Cloud SQL for PostgreSQL Observability

*   `--prebuilt` value: `cloud-sql-postgres-observability`
*   **Permissions:**
    *   **Monitoring Viewer** (`roles/monitoring.viewer`) is required on the
        project to view monitoring data.
*   **Tools:**
    *   `get_system_metrics`: Fetches system level cloud monitoring data
        (timeseries metrics) for a Postgres instance using a PromQL query.
    *   `get_query_metrics`: Fetches query level cloud monitoring data
        (timeseries metrics) for queries running in Postgres instance using a
        PromQL query.
