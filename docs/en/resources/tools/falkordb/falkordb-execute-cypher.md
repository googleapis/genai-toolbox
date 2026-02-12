---
title: "falkordb-execute-cypher"
type: docs
weight: 1
description: >
  A "falkordb-execute-cypher" tool executes any arbitrary Cypher statement against a FalkorDB
  database.
aliases:
- /resources/tools/falkordb-execute-cypher
---

## About

A `falkordb-execute-cypher` tool executes an arbitrary Cypher query provided as a
string parameter against a FalkorDB database. It's designed to be a flexible tool
for interacting with the database when a pre-defined query is not sufficient.
This tool is compatible with any of the following sources:

- [falkordb](../../sources/falkordb.md)

For security, the tool can be configured to be read-only. If the `readOnly` flag
is set to `true`, the tool will analyze the incoming Cypher query and reject any
write operations (like `CREATE`, `MERGE`, `DELETE`, etc.) before execution.

The Cypher query uses standard [Cypher](https://neo4j.com/docs/cypher-manual/current/queries/)
syntax and supports all Cypher features compatible with FalkorDB, including pattern
matching, filtering, and aggregation.

`falkordb-execute-cypher` takes a required input parameter `cypher` and runs the
cypher query against the `source`. It also supports an optional `dry_run`
parameter to validate a query without executing it.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
kind: tools
name: query_falkordb
type: falkordb-execute-cypher
source: my-falkordb-social-graph
readOnly: true
description: |
  Use this tool to execute a Cypher query against the social graph database.
  Only read-only queries are allowed.
  Takes a single 'cypher' parameter containing the full query string.
  Example:
  {{
      "cypher": "MATCH (p:Person {name: 'Alice'})-[:KNOWS]->(friend) RETURN friend.name"
  }}
```

## Reference

| **field**   | **type** | **required** | **description**                                                                                      |
|-------------|:--------:|:------------:|------------------------------------------------------------------------------------------------------|
| type        |  string  |     true     | Must be "falkordb-execute-cypher".                                                                   |
| source      |  string  |     true     | Name of the source the Cypher query should execute on.                                               |
| description |  string  |     true     | Description of the tool that is passed to the LLM.                                                   |
| readOnly    | boolean  |    false     | If set to `true`, the tool will reject any write operations in the Cypher query. Default is `false`. |
