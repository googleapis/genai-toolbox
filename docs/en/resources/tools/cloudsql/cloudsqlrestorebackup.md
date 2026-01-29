---
title: cloud-sql-restore-backup
type: docs
weight: 10
description: "Restores a backup of a Cloud SQL instance."
---

The `cloud-sql-restore-backup` tool restores a backup on a Cloud SQL instance using the Cloud SQL Admin API.

{{< notice info dd>}}
This tool uses a `source` of type `cloud-sql-admin`.
{{< /notice >}}

## Examples

Basic backup restore

```yaml
kind: tools
name: backup-restore-basic
type: cloud-sql-restore-backup
source: cloud-sql-admin-source
description: "Restores a backup onto the given Cloud SQL instance."
```

## Reference

### Tool Configuration
| **field**      | **type** | **required** | **description**                                  |
| -------------- | :------: | :----------: | ------------------------------------------------ |
| type           | string   | true         | Must be "cloud-sql-restore-backup".              |
| source         | string   | true         | The name of the `cloud-sql-admin` source to use. |
| description    | string   | false        | A description of the tool.                       |

### Tool Inputs

| **parameter**     | **type** | **required** | **description**                                                              |
| ------------------| :------: | :----------: | -----------------------------------------------------------------------------|
| target_project    | string   | true         | The project ID of the instance to restore the backup onto.                   |
| target_instance   | string   | true         | The instance to restore the backup onto. Does not include the project ID.    |
| backup_id         | string   | true         | The identifier of the backup being restored.                                 |
| source_project    | string   | false        | (Optional) The project ID of the instance that the backup belongs to.        |
| source_instance   | string   | false        | (Optional) Cloud SQL instance ID of the instance that the backup belongs to. |

## Usage Notes

- The `backup_id` field can be a BackupRun ID (which will be an int64), backup name, or BackupDR backup name.
- If the `backup_id` field contains a BackupRun ID (i.e. an int64), the optional fields `source_project` and `source_instance` must also be provided.

## See Also
- [Cloud SQL Admin API documentation](https://cloud.google.com/sql/docs/mysql/admin-api)
- [Toolbox Cloud SQL tools documentation](../cloudsql)
- [Cloud SQL Restore API documentation](https://cloud.google.com/sql/docs/mysql/backup-recovery/restoring)
