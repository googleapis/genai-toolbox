---
title: "arcadedb-execute-sql Tool"
type: docs
weight: 2
description: >
  Execute SQL queries against ArcadeDB.
---

## About

`arcadedb-execute-sql` executes an arbitrary ArcadeDB SQL statement against an
ArcadeDB source. ArcadeDB supports SQL for document and multi-model queries,
allowing you to query graphs and documents from the same database.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: query_arcadedb_sql
type: arcadedb-execute-sql
source: my-arcadedb-source
description: |
  Execute SQL against ArcadeDB.
  Example:
  {{
      "sql": "SELECT FROM Person WHERE name = :name LIMIT 5",
      "params": {
        "name": "Ada"
      },
      "dry_run": false
  }}
```

## Reference

| **field**   | **type** | **required** | **description**                                                                                |
|-------------|:--------:|:------------:|------------------------------------------------------------------------------------------------|
| type        |  string  |     true     | Must be "arcadedb-execute-sql".                                                                |
| source      |  string  |     true     | Name of the ArcadeDB source the SQL should execute on.                                         |
| description |  string  |     true     | Description of the tool that is passed to the LLM.                                             |
| readOnly    | boolean  |    false     | If true, the statement is routed to a read-only endpoint, and ArcadeDB blocks write statements. |
