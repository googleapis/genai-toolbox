---
title: Cloud SQL List Instance
type: docs
weight: 1
description: "List Cloud SQL instances in a project.\n"
---

The `cloudsql-list-instance` tool lists all Cloud SQL instances in a specified
Google Cloud project.

## Configuration

Here is an example of how to configure the `cloudsql-list-instance` tool in your
`tools.yaml` file:

```yaml
tools:
  list_my_instances:
    kind: cloudsql-list-instance
    description: Use this tool to list all Cloud SQL instances in a project.
```

## Parameters

The `cloudsql-list-instance` tool has one required parameter:

| **field** | **type** | **required** | **description**              |
| --------- | :------: | :----------: | ---------------------------- |
| project   |  string  |     true     | The Google Cloud project ID. |

## Reference

| **field**    |  **type** | **required** | **description**                                                                     |
| ------------ | :-------: | :----------: | ----------------------------------------------------------------------------------- |
| kind         |   string  |     true     | Must be "cloudsql-list-instance".                                                   |
| description  |   string  |     true     | Description of the tool that is passed to the agent.                                |
