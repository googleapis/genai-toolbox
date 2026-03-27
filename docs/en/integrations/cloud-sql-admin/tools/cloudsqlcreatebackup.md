---
title: cloud-sql-create-backup
type: docs
weight: 10
description: "Creates a backup on a Cloud SQL instance."
---

## About

The `cloud-sql-create-backup` tool creates an on-demand backup on a Cloud SQL instance using the Cloud SQL Admin API.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

| **parameter**              | **type** | **required** | **description**                                                                 |
| -------------------------- | :------: | :----------: | ------------------------------------------------------------------------------- |
| project                    | string   | true         | The project ID.                                                                 |
| instance                   | string   | true         | The name of the instance to take a backup on. Does not include the project ID.  |
| location                   | string   | false        | (Optional) Location of the backup run.                                          |
| backup_description         | string   | false        | (Optional) The description of this backup run.                                  |

## Example

Basic backup creation (current state)

```yaml
kind: tool
name: backup-creation-basic
type: cloud-sql-create-backup
source: cloud-sql-admin-source
description: "Creates a backup on the given Cloud SQL instance."
```
## Reference

| **field**      | **type** | **required** | **description**                                               |
| -------------- | :------: | :----------: | ------------------------------------------------------------- |
| type           | string   | true         | Must be "cloud-sql-create-backup".                            |
| source         | string   | true         | The name of the `cloud-sql-admin` source to use.              |
| description    | string   | false        | A description of the tool.                                    |

## Additional Resources
- [Cloud SQL Admin API documentation](https://cloud.google.com/sql/docs/mysql/admin-api)
- [Toolbox Cloud SQL tools documentation](_index.md)
- [Cloud SQL Backup API documentation](https://cloud.google.com/sql/docs/mysql/backup-recovery/backups)