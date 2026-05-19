---
title: cloud-sql-admin-sql-many
type: docs
weight: 1
description: >
  A "cloud-sql-admin-sql-many" tool executes a predefined SQL statement against a specific Cloud SQL instance provided at runtime.
---

## About

The `cloud-sql-admin-sql-many` tool executes a predefined SQL statement against a specific Cloud SQL instance identified by project, instanceId, and database parameters provided at runtime.

It supports both `parameters` and `templateParameters` to allow dynamic values to be injected into the query at runtime.

> **Note:** This tool is intended for developer assistant workflows with human-in-the-loop and shouldn't be used for production agents.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

The following parameters are required at runtime when invoking the tool:

| **Parameter** | **Type** | **Description**            |
| :------------ | :------- | :------------------------- |
| `project`     | string   | The GCP project ID.        |
| `instanceId`  | string   | The Cloud SQL instance ID. |
| `database`    | string   | The database name.         |

Additional parameters may be required based on the `parameters` or `templateParameters` configured in the tool definition.

## Example

```yaml
kind: tool
name: get_user_many_tool
type: cloud-sql-admin-sql-many
source: my-cloud-sql-admin-source
description: Use this tool to get user details from a specific instance.
statement: SELECT * FROM users WHERE id = {{.user_id}}
templateParameters:
  - name: user_id
    type: string
    description: The ID of the user.
```

## Reference

| **field**          | **type** | **required** | **description**                                      |
| :----------------- | :------- | :----------- | :--------------------------------------------------- |
| type               | string   | true         | Must be "cloud-sql-admin-sql-many".                  |
| source             | string   | true         | Name of the `cloud-sql-admin` source.                |
| description        | string   | true         | Description of the tool that is passed to the agent. |
| statement          | string   | true         | The SQL statement template to execute.               |
| parameters         | list     | false        | List of parameters used in the statement template.   |
| templateParameters | list     | false        | List of parameters used in the statement template.   |
