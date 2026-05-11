---
title: "vector-assist-get-spec"
type: docs
weight: 1
description: >
  The "vector-assist-get-spec" tool retrieves the details of an existing vector specification using its unique spec_id.
---

## About

The `vector-assist-get-spec` tool retrieves the details of an existing vector specification using its unique `spec_id`. 

Use this tool to retrieve a vector specification which was created in the context of the vector assist tools. It allows users to inspect the detailed parameters and current state of a particular vector setup. Under the hood, this tool connects to the target database and executes the `vector_assist.get_spec` function.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

{{< notice tip >}} 
Ensure that your target PostgreSQL database has the required `vector_assist` extension installed, in order for this tool to execute successfully.
{{< /notice >}}

## Parameters

The tool takes the following input parameters:

| Parameter | Type   | Description                      | Required |
| :-------- | :----- | :------------------------------- | :------- |
| `spec_id` | string | Unique ID for the vector spec.   | Yes      |

> Note : 
> Parameters are marked as required or optional based on the tool's parameter definitions. 
> The underlying function may perform further validation on optional parameters to ensure 
> all necessary data is available before returning a response.

## Example

```yaml
kind: tool
name: get_spec
type: vector-assist-get-spec
source: my-database-source
description: "This tool retrieves the details of an existing vector specification using its unique 'spec_id'. Use this tool to retrieve a vector specification which was created in the context of the vector assist tools."
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| type        |  string  |     true     | Must be "vector-assist-get-spec".                    |
| source      |  string  |     true     | Name of the source the SQL should execute on.        |
| description |  string  |    false     | Description of the tool that is passed to the agent. |
