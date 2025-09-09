---
title: "cloud-sql-get-instances"
type: docs
weight: 10
description: >
  Get a Cloud SQL instance resource.
---

The `cloud-sql-get-instances` tool retrieves a Cloud SQL instance resource using the Cloud SQL Admin API.

{{< notice info >}}
This tool uses a `source` of kind `http`, and the `baseUrl` for that source must be `https://sqladmin.googleapis.com/`.
{{< /notice >}}

{{< notice info >}}
The toolbox automatically generates a bearer token on behalf of the user with the `https://www.googleapis.com/auth/sqlservice.admin` scope to authenticate requests.
{{< /notice >}}

## Example

```yaml
tools:
  get-sql-instance:
    kind: cloud-sql-get-instances
    description: "Get a Cloud SQL instance resource."
    source: http-source
```

## Reference

| **field**   | **type** | **required** | **description**                                                                                                  |
| ----------- | :------: | :----------: | ---------------------------------------------------------------------------------------------------------------- |
| kind        |  string  |     true     | Must be "cloud-sql-get-instances".                                                                            |
| description |  string  |     true     | A description of the tool.                                                                                       |
| source      |  string  |     true     | The name of the `http` source to use. The source's `baseUrl` must be `https://sqladmin.googleapis.com/`.         |