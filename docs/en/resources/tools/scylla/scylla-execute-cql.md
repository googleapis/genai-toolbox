---
title: "scylla-execute-cql"
type: docs
weight: 1
description: >
  A "scylla-execute-cql" tool executes a CQL statement against a Scylla
  database.
aliases:
- /resources/tools/scylla-execute-cql
---

## About

A `scylla-execute-cql` tool executes a CQL statement against a Scylla
database. It's compatible with any of the following sources:

- [scylla](../../sources/scylla.md)

`scylla-execute-cql` takes one input parameter `cql` and runs the CQL
statement against the `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
 execute_cql_tool:
    kind: scylla-execute-cql
    source: my-scylla-instance
    description: Use this tool to execute CQL statements.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "scylla-execute-cql".                                                                    |
| source      |                   string                   |     true     | Name of the source the CQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
