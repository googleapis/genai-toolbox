---
title: "Cloud SQL for MySQL Observability"
type: docs
description: "Details of the Cloud SQL for MySQL Observability prebuilt configuration."
---

## Cloud SQL for MySQL Observability

*   `--prebuilt` value: `cloud-sql-mysql-observability`
*   **Permissions:**
    *   **Monitoring Viewer** (`roles/monitoring.viewer`) is required on the
        project to view monitoring data.
*   **Tools:**
    *   `get_system_metrics`: Fetches system level cloud monitoring data
        (timeseries metrics) for a MySQL instance using a PromQL query.
    *   `get_query_metrics`: Fetches query level cloud monitoring data
        (timeseries metrics) for queries running in a MySQL instance using a
        PromQL query.
