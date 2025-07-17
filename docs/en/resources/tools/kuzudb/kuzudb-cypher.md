---
title: "kuzudb-cypher"
type: docs
weight: 1
description: > 
  A "kuzudb-cypher" tool executes a pre-defined cypher statement against a KuzuDB
  database.
aliases:
- /resources/tools/kuzudb-cypher
---

## About

A `kuzudb-cypher` tool executes a pre-defined Cypher statement against a KuzuDB graph database. It is designed to work with KuzuDB's embedded graph database, optimized for high query speed and scalability. The tool is compatible with the following sources:

- [kuzudb](../sources/kuzudb.md)

The specified Cypher statement is executed as a [parameterized statement][kuzudb-parameters], with parameters referenced by their name (e.g., `$id`). This approach ensures security by preventing Cypher injection attacks.

> **Note:** This tool uses parameterized queries to prevent SQL injections. \
> Query parameters can be used as substitutes for arbitrary expressions but cannot replace identifiers, node labels, relationship types, or other structural parts of the query.

[kuzudb-parameters]:
    https://docs.kuzudb.com/get-started/prepared-statements/

## Example

```yaml
tools:
  find_collaborators:
    kind: kuzudb-cypher
    source: my-kuzu-social-network
    statement: |
      MATCH (p1:Person)-[:Collaborated_With]->(p2:Person)
      WHERE p1.name = $name AND p2.age > $min_age
      RETURN p2.name, p2.age
      LIMIT 10
    description: |
      Use this tool to find collaborators for a specific person in a social network, filtered by a minimum age.
      Takes a full person name (e.g., "Alice Smith") and a minimum age (e.g., 25) and returns a list of collaborator names and their ages.
      Do NOT use this tool with incomplete names or arbitrary values. Do NOT guess a name or age.
      A person name is a fully qualified name with first and last name separated by a space.
      For example, if given "Smith, Alice" the person name is "Alice Smith".
      If multiple results are returned, prioritize those with the closest collaboration ties.
      Example:
      {{
          "name": "Bob Johnson",
          "min_age": 30
      }}
      Example:
      {{
          "name": "Emma Davis",
          "min_age": 25
      }}
    parameters:
      - name: name
        type: string
        description: Full person name, "firstname lastname"
      - name: min_age
        type: integer
        description: Minimum age as a positive integer
```

## Reference

| **Field**            | **Type**                              | **Required** | **Description**                                                                 |
|----------------------|:-------------------------------------:|:------------:|---------------------------------------------------------------------------------|
| kind                 | string                                | true         | Must be "kuzudb-cypher".                                                       |
| source               | string                                | true         | Name of the KuzuDB source the Cypher query should execute on.                   |
| description          | string                                | true         | Description of the tool that is passed to the LLM for context.                  |
| statement            | string                                | true         | Cypher statement to execute.                                                   |
| authRequired         | []string                              | false        | List of authentication requirements for executing the query (if applicable).    |
| parameters           | [parameters](_index#specifying-parameters) | false    | List of parameters used with the Cypher statement.                              |