---
title: cloud-sql-postgres-pre-check-major-version-upgrade
type: docs
weight: 11
description: Perform a pre-check for a Cloud SQL for PostgreSQL major version upgrade.
---

The `cloud-sql-postgres-precheck-upgrade` tool initiates a pre-check on a Cloud SQL for PostgreSQL
instance to assess its readiness for a major version upgrade using the Cloud SQL Admin API.
It helps identify potential incompatibilities or issues before starting the actual upgrade process.

{{< notice info >}}
This tool uses a `source` of kind `cloud-sql-admin`.
{{< /notice >}}

## Tool Inputs

### Example

```yaml
tools:
  cloud-sql-postgres-pre-check-major-version-upgrade:
    kind: cloud-sql-postgres-pre-check-major-version-upgrade
    source: cloud-sql-admin-source
    description: "Checks if a Cloud SQL PostgreSQL instance is ready for a major version upgrade to the specified target version."
```

### Reference

| **field**    | **type** | **required** | **description**                                           |
| ------------ | :------: | :----------: | --------------------------------------------------------- |
| kind         |  string  |     true     | Must be "cloud-sql-postgres-pre-check-major-version-upgrade". |
| source       |  string  |     true     | The name of the `cloud-sql-admin` source to use.          |
| description  |  string  |     false    | A description of the tool.                                |

| **parameter**           | **type** | **required** | **description**                                                                 |
| ----------------------- | :------: | :----------: | ------------------------------------------------------------------------------- |
| project                 |  string  |     true     | The project ID containing the instance.                                         |
| instance                    |  string  |     true     | The name of the Cloud SQL instance to check.                                    |
| targetDatabaseVersion   |  string  |     false     | The target PostgreSQL major version for the upgrade (e.g., `POSTGRES_18`).  If not specified, defaults to the PostgreSQL 18. |
