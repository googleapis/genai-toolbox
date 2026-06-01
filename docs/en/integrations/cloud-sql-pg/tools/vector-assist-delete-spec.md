---
title: "vector-assist-delete-spec"
type: docs
weight: 1
description: >
  The "vector-assist-delete-spec" tool deletes an existing vector specification and its associated metadata using its spec_id.
---

## About

The `vector-assist-delete-spec` tool deletes an existing vector specification using its `spec_id`. 

Use this tool when a user explicitly requests to delete, remove, or clean up an existing vector specification which was created in the context of the vector assist tools. Under the hood, this tool connects to the target database and executes the `vector_assist.delete_spec` function.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

{{< notice tip >}} 
Ensure that your target PostgreSQL database has the required `vector_assist` extension installed, in order for this tool to execute successfully.
{{< /notice >}}

## Parameters

The tool takes the following input parameters:

| Parameter | Type   | Description                                | Required |
| :-------- | :----- | :----------------------------------------- | :------- |
| `spec_id` | string | Unique ID for the vector spec to delete.   | Yes      |

> Note :
> Parameters are marked as required or optional based on the tool's parameter definitions. 
> The underlying function may perform further validation on optional parameters to ensure 
> all necessary data is available before returning a response.

## Example

```yaml
kind: tool
name: delete_spec
type: vector-assist-delete-spec
source: my-database-source
description: "This tool deletes an existing vector specification using its spec_id. Use this tool when a user explicitly requests to delete, remove, or clean up an existing vector specification which was created in the context of the vector assist tools."
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| type        |  string  |     true     | Must be "vector-assist-delete-spec".                 |
| source      |  string  |     true     | Name of the source the SQL should execute on.        |
| description |  string  |    false     | Description of the tool that is passed to the agent. |