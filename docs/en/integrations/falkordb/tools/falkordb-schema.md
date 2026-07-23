---
title: "falkordb-schema"
type: docs
weight: 3
description: >
  A "falkordb-schema" tool extracts the schema of a FalkorDB graph.
---

## About

A `falkordb-schema` tool extracts a complete schema description of the graph
configured on a FalkorDB source: node labels and relationship types with
their observed property shapes (derived from sampling up to 100 entities per
label or type), indexes (including vector and full-text indexes),
constraints, and graph statistics.

The extracted schema is cached; the `cacheExpireMinutes` field controls the
cache lifetime and defaults to 60 minutes.

## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: get_schema
type: falkordb-schema
source: my-falkordb-instance
description: Use this tool to get the schema of the graph.
cacheExpireMinutes: 10
```

## Output Format

```json
{
  "graphInfo": {"name": "my_graph", "nodeCount": 2, "edgeCount": 1},
  "nodeLabels": [
    {"name": "Person", "count": 1, "properties": [{"name": "name", "types": ["STRING"]}]}
  ],
  "relationships": [
    {"type": "ACTED_IN", "count": 1, "startNode": "Person", "endNode": "Movie", "properties": []}
  ],
  "indexes": [],
  "constraints": [],
  "statistics": {"totalNodes": 2, "totalRelationships": 1}
}
```

## Reference

| **field**          | **type** | **required** | **description**                                                       |
|--------------------|:--------:|:------------:|-----------------------------------------------------------------------|
| type               |  string  |     true     | Must be "falkordb-schema".                                            |
| source             |  string  |     true     | Name of the source to extract the schema from.                        |
| description        |  string  |     true     | Description of the tool that is passed to the LLM.                    |
| cacheExpireMinutes |   int    |    false     | Cache expiration time in minutes. Defaults to 60.                     |
