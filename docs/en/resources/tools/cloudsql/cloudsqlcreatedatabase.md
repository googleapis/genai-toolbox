---
title: cloud-sql-create-database
type: docs
weight: 10
description: >
  Create a new database in a Cloud SQL instance.
---

The `cloud-sql-create-database` tool creates a new database in a specified Cloud SQL instance.

{{< notice info >}}
This tool uses a `source` of kind `cloud-sql-admin`.
{{< /notice >}}

## Example

```yaml
tools:
  create-cloud-sql-database:
    kind: cloud-sql-create-database
    source: my-cloud-sql-admin-source
    description: "Creates a new database in a Cloud SQL instance."
```

## Reference

| **field**   | **type** | **required** | **description**                                  |
| ----------- | :------: | :----------: | ------------------------------------------------ |
| kind        |  string  |     true     | Must be "cloud-sql-create-database".             |
| source      |  string  |     true     | The name of the `cloud-sql-admin` source to use. |
| description |  string  |     false    | A description of the tool.                       |

## Input Parameters

| **parameter** | **type** | **required** | **description**                                                    |
| ------------- | :------: | :----------: | ------------------------------------------------------------------ |
| project       |  string  |     true     | The project ID.                                                    |
| instance      |  string  |     true     | The ID of the instance where the database will be created.         |
| name          |  string  |     true     | The name for the new database. Must be unique within the instance. |
