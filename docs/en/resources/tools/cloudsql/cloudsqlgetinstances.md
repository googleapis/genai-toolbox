---
title: "cloud-sql-get-instance"
type: docs
weight: 10
description: >
  Get a Cloud SQL instance resource.
---

The `cloud-sql-get-instance` tool retrieves a Cloud SQL instance resource using
the Cloud SQL Admin API.

{{< notice info >}}
This tool uses a `source` of kind `cloud-sql-admin`.
{{< /notice >}}

## Example

```yaml
tools:
  get-sql-instance:
    kind: cloud-sql-get-instance
    source: my-cloud-sql-admin-source
    description: "Gets a particular cloud sql instance."
```

## Reference

| **field**   | **type** | **required** | **description**                                  |
| ----------- | :------: | :----------: | ------------------------------------------------ |
| kind        |  string  |     true     | Must be "cloud-sql-get-instance".                |
| source      |  string  |     true     | The name of the `cloud-sql-admin` source to use. |
| description |  string  |     false    | A description of the tool.                       |
