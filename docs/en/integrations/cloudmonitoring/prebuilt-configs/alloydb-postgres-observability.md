---
title: "AlloyDB Postgres Observability"
type: docs
description: "Details of the AlloyDB Postgres Observability prebuilt configuration."
---

## AlloyDB Postgres Observability

*   `--prebuilt` value: `alloydb-postgres-observability`
*   **Permissions:**
    *   **Monitoring Viewer** (`roles/monitoring.viewer`) is required on the
        project to view monitoring data.
*   **Tools:**
    *   `get_system_metrics`: Fetches system level cloud monitoring data
        (timeseries metrics) for an AlloyDB instance using a PromQL query.
    *   `get_query_metrics`: Fetches query level cloud monitoring data
        (timeseries metrics) for queries running in an AlloyDB instance using a
        PromQL query.
