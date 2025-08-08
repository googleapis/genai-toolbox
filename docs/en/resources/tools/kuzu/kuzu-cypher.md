---
title: "kuzu-cypher"
type: docs
weight: 1
description: > 
  A "kuzu-cypher" tool executes a pre-defined cypher statement against a Kuzu
  database.
aliases:
- /resources/tools/kuzu-cypher
---

## About

A `kuzu-cypher` tool executes a pre-defined Cypher statement against a Kuzu graph database. It is designed to work with Kuzu's embedded graph database, optimized for high query speed and scalability. The tool is compatible with the following sources:

- [kuzu](../../sources/kuzu.md)

The specified Cypher statement is executed as a [parameterized statement][kuzu-parameters], with parameters referenced by their name (e.g., `$id`). This approach ensures security by preventing Cypher injection attacks.

> **Note:** This tool uses parameterized queries to prevent Cypher injections. \
> Query parameters can be used as substitutes for arbitrary expressions but cannot replace identifiers, node labels, relationship types, or other structural parts of the query.

[kuzu-parameters]:
    https://docs.kuzudb.com/get-started/prepared-statements/

## Example

```yaml
tools:
  find_collaborators:
    kind: kuzu-cypher
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
### Example with Template Parameters

> **Note:** This tool allows direct modifications to the Cypher statement,
> including identifiers, column names, and table names. **This makes it more
> vulnerable to Cypher injections**. Using basic parameters only (see above) is
> recommended for performance and safety reasons. For more details, please check
> [templateParameters](../#template-parameters).

```yaml
tools:
  find_friends:
    kind: kuzu-cypher
    source: my-kuzu-social-network
    statement: |
      MATCH (p1:{{.nodeLabel}})-[:friends_with]->(p2:{{.nodeLabel}})
      WHERE p1.name = $name
      RETURN p2.name
      LIMIT 5
    description: |
      Use this tool to find friends of a specific person in a social network.
      Takes a node label (e.g., "Person") and a full person name (e.g., "Alice Smith") and returns a list of friend names.
      Do NOT use with incomplete names. A person name is a full name with first and last name separated by a space.
      Example:
      {
        "nodeLabel": "Person",
        "name": "Bob Johnson"
      }
    templateParameters:
      - name: nodeLabel
        type: string
        description: Node label for the table to query, e.g., "Person"
    parameters:
      - name: name
        type: string
        description: Full person name, "firstname lastname"
```

## Reference

| **Field**            | **Type**                              | **Required** | **Description**                                                                 |
|----------------------|:-------------------------------------:|:------------:|---------------------------------------------------------------------------------|
| kind                 | string                                | true         | Must be "kuzu-cypher".                                                       |
| source               | string                                | true         | Name of the Kuzu source the Cypher query should execute on.                   |
| description          | string                                | true         | Description of the tool that is passed to the LLM for context.                  |
| statement            | string                                | true         | Cypher statement to execute.                                                   |
| authRequired         | []string                              | false        | List of authentication requirements for executing the query (if applicable).    |
| parameters           | [parameters](../#specifying-parameters) | false    | List of parameters used with the Cypher statement.                              |
| templateParameters | [templateParameters](../#template-parameters) |    false     | List of [templateParameters](../#template-parameters) that will be inserted into the Cypher statement before executing prepared statement. |