---
title: Cloud SQL List Instances
type: docs
weight: 1
description: "List Cloud SQL instances in a project.\n"
---

The `cloud-sql-list-instances` tool lists all Cloud SQL instances in a specified
Google Cloud project.

{{< notice info >}}
This tool uses the `cloud-sql-admin` source, which automatically handles
authentication on behalf of the user.
{{< /notice >}}

## Configuration

Here is an example of how to configure the `cloud-sql-list-instances` tool in
your `tools.yaml` file:

```yaml
sources:
  my-cloud-sql-admin-source:
    kind: cloud-sql-admin

tools:
  list_my_instances:
    kind: cloud-sql-list-instances
    source: my-cloud-sql-admin-source
    description: Use this tool to list all Cloud SQL instances in a project.
```

## Parameters

The `cloud-sql-list-instances` tool has one required parameter:

| **field** | **type** | **required** | **description**              |
| --------- | :------: | :----------: | ---------------------------- |
| project   |  string  |     true     | The Google Cloud project ID. |

## Reference

| **field**   | **type** | **required** | **description**                                                |
|-------------|:--------:|:------------:|----------------------------------------------------------------|
| kind        |  string  |     true     | Must be "cloud-sql-list-instances".                            |
| description |  string  |    false     | Description of the tool that is passed to the agent.           |
| source      |  string  |     true     | The name of the `cloud-sql-admin` source to use for this tool. |
