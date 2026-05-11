---
title: "vector-assist-list-specs"
type: docs
weight: 1
description: >
  The "vector-assist-list-specs" tool retrieves a list of all defined vector specifications for a given table and column.
---

## About

The `vector-assist-list-specs` tool lists all defined vector specifications for a given table and column name. 

Use this tool to list vector specifications which were created in the context of the vector assist tools. It provides a high-level overview of existing vector setups. Under the hood, this tool connects to the target database and executes the `vector_assist.list_specs` function.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

{{< notice tip >}} 
Ensure that your target PostgreSQL database has the required `vector_assist` extension installed, in order for this tool to execute successfully.
{{< /notice >}}

## Parameters

The tool takes the following input parameters:

| Parameter     | Type   | Description                                          | Required |
| :------------ | :----- | :--------------------------------------------------- | :------- |
| `table_name`  | string | Table name to list vector specifications for.        | Yes      |
| `column_name` | string | Column name to list vector specifications for.       | No       |

> Note : 
> Parameters are marked as required or optional based on the tool's parameter definitions. 
> The underlying function may perform further validation on optional parameters to ensure 
> all necessary data is available before returning a response.

## Example

```yaml
kind: tool
name: list_specs
type: vector-assist-list-specs
source: my-database-source
description: "This tool lists all defined vector specifications for a given table and column name. Use this tool to list vector specifications which were created in the context of the vector assist tools."
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| type        |  string  |     true     | Must be "vector-assist-list-specs".                  |
| source      |  string  |     true     | Name of the source the SQL should execute on.        |
| description |  string  |    false     | Description of the tool that is passed to the agent. |