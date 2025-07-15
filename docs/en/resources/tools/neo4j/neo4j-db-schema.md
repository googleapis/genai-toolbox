---
title: "neo4j-db-schema"
type: docs
weight: 1
description: > 
  A "neo4j-db-schema" tool extracts a comprehensive schema from a Neo4j
  database.
aliases:
- /resources/tools/neo4j-db-schema
---

## About

A `neo4j-db-schema` tool connects to a Neo4j database and extracts its complete schema information. It runs multiple queries concurrently to efficiently gather details about node labels, relationships, properties, constraints, and indexes. This includes using procedures like `apoc.meta.schema` to get a detailed overview of the database structure.

This tool takes no parameters and is compatible with any of the following sources:

- [neo4j](../sources/neo4j.md)

The output is a structured JSON object containing all the schema details, which can be invaluable for providing database context to an LLM.

## Example

```yaml
tools:
  get_movie_db_schema:
    kind: neo4j-db-schema
    source: my-neo4j-movies-instance
    description: |
      Use this tool to get the full schema of the movie database.
      This provides information on all available node labels (like Movie, Person), 
      relationships (like ACTED_IN), and the properties on each.
      This tool takes no parameters.
```

## Reference
| **field**     |                  **type**                  | **required** | **description**                                                                                 |
|---------------|:------------------------------------------:|:------------:|-------------------------------------------------------------------------------------------------|
| kind          |                   string                   |     true     | Must be "neo4j-db-schema".                                                                      |
| source        |                   string                   |     true     | Name of the source the schema should be extracted from.                                         |
| description   |                   string                   |     true     | Description of the tool that is passed to the LLM.                                              |
