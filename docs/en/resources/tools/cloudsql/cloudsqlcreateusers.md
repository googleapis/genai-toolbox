---
title: cloud-sql-create-users
type: docs
weight: 10
description: >
  Create a new user in a Cloud SQL instance.
---

The `cloud-sql-create-users` tool creates a new user in a specified Cloud SQL
instance. It can create both built-in and IAM users.

{{< notice info >}}
This tool uses a `source` of kind `cloud-sql-admin`.
{{< /notice >}}

## Example

```yaml
tools:
  create-cloud-sql-user:
    kind: cloud-sql-create-users
    source: my-cloud-sql-admin-source
    description: "Creates a new user in a Cloud SQL instance. Both built-in and IAM users are supported. IAM users require an email account as the user name. IAM is the more secure and recommended way to manage users. The agent should always ask the user what type of user they want to create. For more information, see https://cloud.google.com/sql/docs/postgres/add-manage-iam-users"
```

## Reference

| **field**    |  **type** | **required** | **description**                                  |
| ------------ | :-------: | :----------: | ------------------------------------------------ |
| kind         |   string  |     true     | Must be "cloud-sql-create-users".                |
| description  |   string  |     false    | A description of the tool.                       |
| source       |   string  |     true     | The name of the `cloud-sql-admin` source to use. |
