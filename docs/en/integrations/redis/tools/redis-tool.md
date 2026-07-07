---
title: "redis"
type: docs
weight: 1
description: >
  A "redis" tool executes a set of pre-defined Redis commands against a Redis instance.

---

## About

A redis tool executes a series of pre-defined Redis commands against a
Redis source.

The specified Redis commands are executed sequentially. Each command is
represented as a string list, where the first element is the command name (e.g.,
SET, GET, HGETALL) and subsequent elements are its arguments.

### Dynamic Command Parameters

Command arguments can be templated using the `$variableName` annotation. The
array type parameters will be expanded once into multiple arguments. Take the
following config for example:

```yaml
  commands:
      - [SADD, userNames, $userNames] # Array will be flattened into multiple arguments.
  parameters:
    - name: userNames
      type: array
      description: The user names to be set.  
      items:
          name: userName # the item name doesn't matter but it has to exist
          type: string 
          description: username
```

If the input is an array of strings `["Alice", "Sid", "Bob"]`,  The final command
to be executed after argument expansion will be `[SADD, userNames, Alice, Sid, Bob]`.

## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: user_data_tool
type: redis
source: my-redis-instance
description: |
  Use this tool to interact with user data stored in Redis.
  It can set, retrieve, and delete user-specific information.
commands:
  - [SADD, userNames, $userNames] # Array will be flattened into multiple arguments.
  - [GET, $userId]
parameters:
  - name: userId
    type: string
    description: The unique identifier for the user.
  - name: userNames
    type: array
    description: The user names to be set.  
```

### Vector Search

Redis supports vector similarity search (using RediSearch). When using an `embeddingModel` with a `redis` tool, the tool automatically converts text parameters into the little-endian binary vector format required by Redis.

#### Vector Ingestion Example

This tool stores a text content and its vector representation as a binary string in a Redis hash.

```yaml
kind: tool
name: insert_doc_redis
type: redis
source: my-redis-source
commands:
  - ["HSET", "doc:$id", "content", "$content", "embedding", "$text_to_embed"]
description: |
  Index new documents for semantic search in Redis.
parameters:
  - name: id
    type: string
    description: The unique ID of the document.
  - name: content
    type: string
    description: The text content to store.
  - name: text_to_embed
    type: string
    valueFromParam: content
    embeddedBy: gemini-model
```

#### Vector Search Example

This tool performs a semantic search using `FT.SEARCH` with KNN vector similarity. The search query string provided by the Agent is converted into a binary vector before the search command is executed.

```yaml
kind: tool
name: search_docs_redis
type: redis
source: my-redis-source
commands:
  - ["FT.SEARCH", "idx:docs", "(*)=>[KNN 5 @embedding $query]", "PARAMS", "2", "query", "$query", "DIALECT", "2"]
description: |
  Search for documents in Redis using natural language.
parameters:
  - name: query
    type: string
    description: The search query.
    embeddedBy: gemini-model
```
