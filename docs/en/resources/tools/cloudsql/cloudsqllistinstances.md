---
title: Cloud SQL List Instance
type: docs
weight: 1
description: "List Cloud SQL instances in a project.\n"
---

The `cloud-sql-list-instances` tool lists all Cloud SQL instances in a specified
Google Cloud project.

{{< notice info >}}
The toolbox automatically generates a bearer token on behalf of the user with the `https://www.googleapis.com/auth/sqlservice.admin` scope to authenticate requests.
{{< /notice >}}

## Configuration

Here is an example of how to configure the `cloud-sql-list-instances` tool in your
`tools.yaml` file:

```yaml
sources:
  my_http_source:
    kind: http
    baseUrl: https://sqladmin.googleapis.com

tools:
  list_my_instances:
    kind: cloud-sql-list-instances
    description: Use this tool to list all Cloud SQL instances in a project.
    source: my_http_source
```

## Parameters

The `cloud-sql-list-instances` tool has one required parameter:

| **field** | **type** | **required** | **description**              |
| --------- | :------: | :----------: | ---------------------------- |
| project   |  string  |     true     | The Google Cloud project ID. |

## Reference

| **field**    |  **type** | **required** | **description**                                                                     |
| ------------ | :-------: | :----------: | ----------------------------------------------------------------------------------- |
| kind         |   string  |     true     | Must be "cloud-sql-list-instances".                                                 |
| description  |   string  |     true     | Description of the tool that is passed to the agent.                                |
| source       |   string  |     true     | The name of the `http` source to use for this tool.                                 |
