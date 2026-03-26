---
title: "Cloud SQL for SQL Server Observability"
type: docs
description: "Details of the Cloud SQL for SQL Server Observability prebuilt configuration."
---

## Cloud SQL for SQL Server Observability

*   `--prebuilt` value: `cloud-sql-mssql-observability`
*   **Permissions:**
    *   **Monitoring Viewer** (`roles/monitoring.viewer`) is required on the
        project to view monitoring data.
*   **Tools:**
    *   `get_system_metrics`: Fetches system level cloud monitoring data
        (timeseries metrics) for a SQL Server instance using a PromQL query.
