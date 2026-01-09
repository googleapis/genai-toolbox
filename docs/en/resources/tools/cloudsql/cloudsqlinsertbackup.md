---
title: cloud-sql-create-backup
type: docs
weight: 10
description: "Creates a backup on a Cloud SQL instance."
---

The `cloud-sql-create-backup` tool creates an on-demand backup on a Cloud SQL instance using the Cloud SQL Admin API.

{{< notice info dd>}}
This tool uses a `source` of kind `cloud-sql-admin`.
{{< /notice >}}

## Examples

Basic backup creation (current state)

```yaml
tools:
  backup-creation-basic:
    kind: cloud-sql-create-backup
    source: cloud-sql-admin-source
    description: "Creates a backup on the given Cloud SQL instance."
```
## Reference
### Tool Configuration
| **field**      | **type** | **required** | **description**                                               |
| -------------- | :------: | :----------: | ------------------------------------------------------------- |
| kind           | string   | true         | Must be "cloud-sql-create-backup".                            |
| source         | string   | true         | The name of the `cloud-sql-admin` source to use.              |
| description    | string   | false        | A description of the tool.                                    |

### Tool Inputs

| **parameter**              | **type** | **required** | **description**                                                                 |
| -------------------------- | :------: | :----------: | ------------------------------------------------------------------------------- |
| project                    | string   | true         | The project ID.                                                                 |
| instance                   | string   | true         | The name of the instance to take a backup on. Does not include the project ID.  |
| location                   | string   | false        | (Optional) Location of the backup run.                                          |
| description                | string   | false        | (Optional) The description of this backup run.                                  |

## Usage Notes

- The tool is used for taking on-demand backups on Cloud SQL instances.
- The source must be a valid Cloud SQL Admin API source.
- You can optionally specify the `location` parameter to set the location of the backup. If omitted, the backup will be stored in the custom backup location if set on the instance, or the multi-region that is geographically closest to the location of the Cloud SQL instance if not set.
- You can optionally specify the `description` parameter to set a description for the backup. If omitted, the description will be empty.

## See Also
- [Cloud SQL Admin API documentation](https://cloud.google.com/sql/docs/mysql/admin-api)
- [Toolbox Cloud SQL tools documentation](../cloudsql)
- [Cloud SQL Backup API documentation](https://cloud.google.com/sql/docs/mysql/backup-recovery/backups)