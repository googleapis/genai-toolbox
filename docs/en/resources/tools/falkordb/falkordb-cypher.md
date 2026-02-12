---
title: "falkordb-cypher"
type: docs
weight: 1
description: >
  A "falkordb-cypher" tool executes a pre-defined cypher statement against a FalkorDB
  database.
aliases:
- /resources/tools/falkordb-cypher
---

## About

A `falkordb-cypher` tool executes a pre-defined Cypher statement against a FalkorDB
database. It's compatible with any of the following sources:

- [falkordb](../../sources/falkordb.md)

The specified Cypher statement is executed as a parameterized statement, and
specified parameters will be used according to their name: e.g. `$name`.

> **Note:** This tool uses parameterized queries to prevent Cypher injection attacks.
> Query parameters can be used as substitutes for arbitrary expressions.
> Parameters cannot be used as substitutes for identifiers, labels, relationship
> types, property names, or other parts of the query.

## Example

```yaml
kind: tools
name: create_person
type: falkordb-cypher
source: my-falkordb-instance
statement: |
  CREATE (p:Person {name: $name, age: $age})
  RETURN p.name, p.age
description: |
  Use this tool to create a new person node in the social graph.
  Takes a person's name and age and creates a Person node with those properties.
  Returns the name and age of the created person.
  Example:
  {{
      "name": "Alice",
      "age": 30
  }}
parameters:
  - name: name
    type: string
    description: Full name of the person
  - name: age
    type: integer
    description: Age of the person
```

## Reference

| **field**   |                **type**                 | **required** | **description**                                                                              |
|-------------|:---------------------------------------:|:------------:|----------------------------------------------------------------------------------------------|
| type        |                 string                  |     true     | Must be "falkordb-cypher".                                                                   |
| source      |                 string                  |     true     | Name of the source the Cypher query should execute on.                                       |
| description |                 string                  |     true     | Description of the tool that is passed to the LLM.                                           |
| statement   |                 string                  |     true     | Cypher statement to execute                                                                  |
| parameters  | [parameters](../#specifying-parameters) |    false     | List of [parameters](../#specifying-parameters) that will be used with the Cypher statement. |
