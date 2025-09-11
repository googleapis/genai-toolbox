---
title: "cloud-sql-create-users"
type: docs
weight: 10
description: >
  Create a new user in a Cloud SQL instance.
---

The `cloud-sql-create-users` tool creates a new user in a specified Cloud SQL instance. It can create both built-in and IAM users.

{{< notice info >}}
This tool uses a `source` of kind `cloud-sql-admin`.
{{< /notice >}}

## Example

```yaml
tools:
  create-cloud-sql-user:
    kind: cloud-sql-create-users
    description: "Creates a new user in a Cloud SQL instance."
    source: my-cloud-sql-source
```

## Reference

| **field**   | **type** | **required** | **description**                                                                                                  |
| ----------- | :------: | :----------: | ---------------------------------------------------------------------------------------------------------------- |
| kind        |  string  |     true     | Must be "cloud-sql-create-users".                                                                            |
| description |  string  |    true      | A description of the tool.                                                                                       |
| source      |  string  |    true      | The name of the `cloud-sql-admin` source to use.                                                                 |
| authRequired| []string |    false     | A list of auth services required by the tool.                                                                    |
