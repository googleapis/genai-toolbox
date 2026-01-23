---
title: "Neo4j"
type: docs
weight: 1
description: >
  Neo4j is a powerful, open source graph database system

---

## About

[Neo4j][neo4j-docs] is a powerful, open source graph database system with over
15 years of active development that has earned it a strong reputation for
reliability, feature robustness, and performance.

[neo4j-docs]: https://neo4j.com/docs

## Available Tools

- [`neo4j-cypher`](../tools/neo4j/neo4j-cypher.md)  
  Run Cypher queries against your Neo4j graph database.

## Requirements

### Database User

This source only uses standard authentication. You will need to [create a Neo4j
user][neo4j-users] to log in to the database with, or use the default `neo4j`
user if available.

[neo4j-users]: https://neo4j.com/docs/operations-manual/current/authentication-authorization/manage-users/

## Example

```yaml
sources:
    my-neo4j-source:
        kind: neo4j
        uri: neo4j+s://xxxx.databases.neo4j.io:7687
        user: ${USER_NAME}
        password: ${PASSWORD}
        database: "neo4j"
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field** | **type** | **required** | **description**                                                      |
|-----------|:--------:|:------------:|----------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "neo4j".                                                     |
| uri       |  string  |     true     | Connect URI ("bolt://localhost", "neo4j+s://xxx.databases.neo4j.io") |
| user      |  string  |     true     | Name of the Neo4j user to connect as (e.g. "neo4j").                 |
| password  |  string  |     true     | Password of the Neo4j user (e.g. "my-password").                     |
| database  |  string  |     true     | Name of the Neo4j database to connect to (e.g. "neo4j").             |
