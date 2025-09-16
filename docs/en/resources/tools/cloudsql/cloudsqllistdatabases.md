---
title: cloud-sql-list-databases
type: docs
weight: 1
description: List Cloud SQL databases in an instance.
---

The `cloud-sql-list-databases` tool lists all Cloud SQL databases in a specified
Google Cloud project and instance.

{{< notice info >}}
This tool uses the `cloud-sql-admin` source.
{{< /notice >}}

## Configuration

Here is an example of how to configure the `cloud-sql-list-databases` tool in your
`tools.yaml` file:

```yaml
sources:
  my-cloud-sql-admin-source:
    kind: cloud-sql-admin

tools:
  list_my_databases:
    kind: cloud-sql-list-databases
    source: my-cloud-sql-admin-source
    description: Use this tool to list all Cloud SQL databases in an instance.
```

## Parameters

The `cloud-sql-list-databases` tool has two required parameters:

| **field** | **type** | **required** | **description**              |
| --------- | :------: | :----------: | ---------------------------- |
| project   |  string  |     true     | The Google Cloud project ID. |
| instance  |  string  |     true     | The Cloud SQL instance ID.   |

## Reference

| **field**   | **type** | **required** | **description**                                                |
| ----------- | :------: | :----------: | -------------------------------------------------------------- |
| kind        |  string  |     true     | Must be "cloud-sql-list-databases".                            |
| source      |  string  |     true     | The name of the `cloud-sql-admin` source to use for this tool. |
| description |  string  |     false    | Description of the tool that is passed to the agent.           |
