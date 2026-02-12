---
title: "FalkorDB"
type: docs
weight: 1
description: >
  FalkorDB is a high-performance graph database built on Redis

---

## About

[FalkorDB][falkordb-docs] is a high-performance graph database built on Redis
that uses Cypher as its query language (the same query language as Neo4j).
FalkorDB leverages Redis's speed and efficiency to provide ultra-fast graph
operations while maintaining compatibility with standard Cypher syntax.

[falkordb-docs]: https://docs.falkordb.com

## Available Tools

- [`falkordb-cypher`](../tools/falkordb/falkordb-cypher.md)
  Run pre-defined Cypher queries against your FalkorDB graph database.

- [`falkordb-execute-cypher`](../tools/falkordb/falkordb-execute-cypher.md)
  Execute arbitrary Cypher queries against your FalkorDB graph database.

## Requirements

### Redis Connection

FalkorDB runs as a Redis module, so you'll need:
- A running FalkorDB instance (which is Redis with the FalkorDB module loaded)
- Connection details (address and port, typically `localhost:6379`)
- Optional authentication credentials (username and password)

## Example

```yaml
kind: sources
name: my-falkordb-source
type: falkordb
addr: localhost:6379
graph: social
username: ${FALKORDB_USERNAME}
password: ${FALKORDB_PASSWORD}
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field** | **type** | **required** | **description**                                                  |
|-----------|:--------:|:------------:|------------------------------------------------------------------|
| type      |  string  |     true     | Must be "falkordb".                                              |
| addr      |  string  |     true     | Redis address (e.g. "localhost:6379").                           |
| graph     |  string  |     true     | Name of the FalkorDB graph to connect to (e.g. "social").        |
| username  |  string  |    false     | Redis username for authentication (optional).                    |
| password  |  string  |    false     | Redis password for authentication (optional).                    |
