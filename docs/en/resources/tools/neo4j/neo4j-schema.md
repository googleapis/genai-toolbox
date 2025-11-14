---
title: "neo4j-schema"
type: "docs"
weight: 1
description: > 
  A "neo4j-schema" tool extracts a comprehensive schema from a Neo4j
  database.
aliases:
- /resources/tools/neo4j-schema
---

## About

A `neo4j-schema` tool connects to a Neo4j database and extracts its complete
schema information. It runs multiple queries concurrently to efficiently gather
details about node labels, relationships, properties, constraints, and indexes.

The tool automatically detects if the APOC (Awesome Procedures on Cypher)
library is available. If so, it uses APOC procedures like `apoc.meta.schema` for
a highly detailed overview of the database structure; otherwise, it falls back
to using native Cypher queries.

The extracted schema is **cached** to improve performance for subsequent
requests. The output is a structured JSON object containing all the schema
details, which can be invaluable for providing database context to an LLM. This
tool is compatible with a `neo4j` source and takes no parameters.

## Example

```yaml
tools:
  get_movie_db_schema:
    kind: neo4j-schema
    source: my-neo4j-movies-instance
    description: |
      Use this tool to get the full schema of the movie database.
      This provides information on all available node labels (like Movie, Person), 
      relationships (like ACTED_IN), and the properties on each.
      This tool takes no parameters.
    # Optional configuration to cache the schema for 2 hours
    cacheExpireMinutes: 120
```

## Reference

| **field**          | **type** | **required** | **description**                                         |
|--------------------|:--------:|:------------:|---------------------------------------------------------|
| kind               |  string  |     true     | Must be `neo4j-schema`.                                 |
| source             |  string  |     true     | Name of the source the schema should be extracted from. |
| description        |  string  |     true     | Description of the tool that is passed to the LLM.      |
| cacheExpireMinutes | integer  |    false     | Cache expiration time in minutes. Defaults to 60.       |
