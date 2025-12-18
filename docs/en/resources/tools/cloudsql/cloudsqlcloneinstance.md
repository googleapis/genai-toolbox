---
title: cloud-sql-clone-instance
type: docs
weight: 10
description: "Clone a Cloud SQL instance."
---

The `cloud-sql-clone-instance` tool clones a Cloud SQL instance using the Cloud SQL Admin API.

{{< notice info dd>}}
This tool uses a `source` of kind `cloud-sql-admin`.
{{< /notice >}}

## Examples

Basic clone (current state)

```yaml
tools:
  clone-instance-basic:
    kind: cloud-sql-clone-instance
    source: cloud-sql-admin-source
    description: "Creates an exact copy of a Cloud SQL instance. Supports configuring instance zones and high-availability setup through zone preferences."
```

Point-in-time recovery (PITR) clone

```yaml
tools:
  clone-instance-pitr:
    kind: cloud-sql-clone-instance
    source: cloud-sql-admin-source
    description: "Creates an exact copy of a Cloud SQL instance at a specific point in time (PITR). Supports configuring instance zones and high-availability setup through zone preferences"
```

## Reference

### Tool Configuration

| **field**      | **type** | **required** | **description**                                               |
| -------------- | :------: | :----------: | ------------------------------------------------------------- |
| kind           | string   | true         | Must be "cloud-sql-clone-instance".                           |
| source         | string   | true         | The name of the `cloud-sql-admin` source to use.              |
| description    | string   | false        | A description of the tool.                                    |

### Tool Inputs

| **parameter**              | **type** | **required** | **description**                                                                 |
| -------------------------- | :------: | :----------: | ------------------------------------------------------------------------------- |
| project                    | string   | true         | The project ID.                                                                 |
| sourceInstanceName         | string   | true         | The name of the source instance to clone.                                       |
| destinationInstanceName    | string   | true         | The name of the new (cloned) instance.                                          |
| pointInTime                | string   | false        | (Optional) The point in time for a PITR (Point-In-Time Recovery) clone.         |
| preferredZone              | string   | false        | (Optional) The preferred zone for the cloned instance. If not specified, defaults to the source instance's zone. |
| preferredSecondaryZone     | string   | false        | (Optional) The preferred secondary zone for the cloned instance (for HA). |

## Usage Notes

- The tool supports both basic clone and point-in-time recovery (PITR) clone operations.
- For PITR, specify the `pointInTime` parameter in RFC3339 format (e.g., `2024-01-01T00:00:00Z`).
- The source must be a valid Cloud SQL Admin API source.
- You can optionally specify the `zone` parameter to set the zone for the cloned instance. If omitted, the zone of the source instance will be used.
- You can optionally specify the `preferredZone` and `preferredSecondaryZone` (only in REGIONAL instances) to set the preferred zones for the cloned instance. These are useful for high availability (HA) configurations. If omitted, defaults will be used based on the source instance.

## See Also
- [Cloud SQL Admin API documentation](https://cloud.google.com/sql/docs/mysql/admin-api)
- [Toolbox Cloud SQL tools documentation](../cloudsql)
- [Cloud SQL Clone API documentation](https://cloud.google.com/sql/docs/mysql/clone-instance)