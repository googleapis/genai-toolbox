---
title: cloud-sql-list-instances
type: docs
weight: 1
description: "List Cloud SQL instances in a project.\n"
---

## About

The `cloud-sql-list-instances` tool lists all Cloud SQL instances in a specified
Google Cloud project.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

The `cloud-sql-list-instances` tool has one required parameter:

| **field** | **type** | **required** | **description**              |
| --------- | :------: | :----------: | ---------------------------- |
| project   |  string  |     true     | The Google Cloud project ID. |

## Example

Here is an example of how to configure the `cloud-sql-list-instances` tool in
your `tools.yaml` file:

```yaml
kind: source
name: my-cloud-sql-admin-source
type: cloud-sql-admin
---
kind: tool
name: list_my_instances
type: cloud-sql-list-instances
source: my-cloud-sql-admin-source
description: Use this tool to list all Cloud SQL instances in a project.
```

## Reference

| **field**   | **type** | **required** | **description**                                                |
|-------------|:--------:|:------------:|----------------------------------------------------------------|
| type        |  string  |     true     | Must be "cloud-sql-list-instances".                            |
| description |  string  |    false     | Description of the tool that is passed to the agent.           |
| source      |  string  |     true     | The name of the `cloud-sql-admin` source to use for this tool. |
