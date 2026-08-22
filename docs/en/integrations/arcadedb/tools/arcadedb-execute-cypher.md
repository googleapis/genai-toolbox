---
title: "arcadedb-execute-cypher Tool"
type: docs
weight: 1
description: >
  Execute Cypher queries against ArcadeDB via Bolt.
---

## About

`arcadedb-execute-cypher` executes an arbitrary Cypher query against an
ArcadeDB source over the Bolt protocol. It supports a `readOnly` mode that
rejects write statements and a `dry_run` mode that validates queries without
executing them.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: query_arcadedb
type: arcadedb-execute-cypher
source: my-arcadedb-source
readOnly: true
description: |
  Execute Cypher against ArcadeDB in read-only mode.
  Example:
  {{
      "cypher": "MATCH (n) RETURN count(n)"
  }}
```

## Reference

| **field**   | **type** | **required** | **description**                                                                                      |
|-------------|:--------:|:------------:|------------------------------------------------------------------------------------------------------|
| type        |  string  |     true     | Must be "arcadedb-execute-cypher".                                                                  |
| source      |  string  |     true     | Name of the ArcadeDB source the Cypher query should execute on.                                      |
| description |  string  |     true     | Description of the tool that is passed to the LLM.                                                   |
| readOnly    | boolean  |    false     | If set to `true`, the tool will reject any write operations in the Cypher query. Default is `false`. |
