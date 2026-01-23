---
title: "EmbeddingModels"
type: docs
weight: 2
description: >
  EmbeddingModels represent services that transform text into vector embeddings for semantic search.
---

EmbeddingModels represent services that generate vector representations of text
data. In the MCP Toolbox, these models enable **Semantic Queries**,
allowing [Tools](../tools/) to automatically convert human-readable text into
numerical vectors before using them in a query.

This is primarily used in two scenarios:

- **Vector Ingestion**: Converting a text parameter into a vector string during
  an `INSERT` operation.

- **Semantic Search**: Converting a natural language query into a vector to
  perform similarity searches.

## Example

The following configuration defines an embedding model and applies it to
specific tool parameters.

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your API keys into the configuration file.
{{< /notice >}}

### Step 1 - Define an Embedding Model

Define an embedding model in the `embeddingModels` section:

```yaml
embeddingModels:
  gemini-model: # Name of the embedding model
    kind: gemini
    model: gemini-embedding-001
    apiKey: ${GOOGLE_API_KEY}
    dimension: 768

```

### Step 2 - Embed Tool Parameters

Use the defined embedding model, embed your query parameters using the
`embeddedBy` field. Only string-typed
parameters can be embedded:

```yaml
tools:
  # Vector ingestion tool
  insert_embedding:
    kind: postgres-sql
    source: my-pg-instance
    statement: |
      INSERT INTO documents (content, embedding) 
      VALUES ($1, $2);
    parameters:
      - name: content
        type: string
      - name: vector_string
        type: string
        description: The text to be vectorized and stored.
        embeddedBy: gemini-model # refers to the name of a defined embedding model

  # Semantic search tool
  search_embedding:
    kind: postgres-sql
    source: my-pg-instance
    statement: |
      SELECT id, content, embedding <-> $1 AS distance 
      FROM documents
      ORDER BY distance LIMIT 1
    parameters:
      - name: semantic_search_string
        type: string
        description: The search query that will be converted to a vector.
        embeddedBy: gemini-model # refers to the name of a defined embedding model
```

## Kinds of Embedding Models
