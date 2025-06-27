---
title: "neo4j-cypher"
type: docs
weight: 1
description: > 
  A "neo4j-cypher" tool executes a pre-defined cypher statement against a Neo4j
  database.
aliases:
- /resources/tools/neo4j-cypher
---

## About

A `neo4j-cypher` tool executes a pre-defined Cypher statement against a Neo4j
database. It's compatible with any of the following sources:

- [neo4j](../sources/neo4j.md)

The specified Cypher statement is executed as a [parameterized
statement][neo4j-parameters], and specified parameters will be used according to
their name: e.g. `$id`.

> **Note:** This tool uses parameterized queries to prevent SQL injections.
> Query parameters can be used as substitutes for arbitrary expressions.
> Parameters cannot be used as substitutes for identifiers, column names, table
> names, or other parts of the query.

[neo4j-parameters]:
    https://neo4j.com/docs/cypher-manual/current/syntax/parameters/

## Example

```yaml
tools:
 search_movies_by_actor:
    kind: neo4j-cypher
    source: my-neo4j-movies-instance
    statement: |
      MATCH (m:Movie)<-[:ACTED_IN]-(p:Person)
      WHERE p.name = $name AND m.year > $year
      RETURN m.title, m.year
      LIMIT 10
    description: |
      Use this tool to get a list of movies for a specific actor and a given minimum release year.
      Takes an full actor name, e.g. "Tom Hanks" and a year e.g 1993 and returns a list of movie titles and release years.
      Do NOT use this tool with a movie title. Do NOT guess an actor name, Do NOT guess a year.
      A actor name is a fully qualified name with first and last name separated by a space.
      For example, if given "Hanks, Tom" the actor name is "Tom Hanks".
      If the tool returns more than one option choose the most recent movies.
      Example:
      {{
          "name": "Meg Ryan",
          "year": 1993
      }}
      Example:
      {{
          "name": "Clint Eastwood",
          "year": 2000
      }}
    parameters:
      - name: name
        type: string
        description: Full actor name, "firstname lastname"
      - name: year
        type: integer
        description: 4 digit number starting in 1900 up to the current year 
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                 |
|-------------|:------------------------------------------:|:------------:|-------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "neo4j-cypher".                                                                         |
| source      |                   string                   |     true     | Name of the source the Cypher query should execute on.                                          |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                              |
| statement   |                   string                   |     true     | Cypher statement to execute                                                                     |
| parameters  | [parameters](_index#specifying-parameters) |    false     | List of [parameters](_index#specifying-parameters) that will be used with the Cypher statement. |
