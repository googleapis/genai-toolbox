---
title: "mongodb-insert-many"
type: docs
weight: 1
description: > 
  A "mongodb-insert-many" tool inserts multiple new documents into a MongoDB collection.
aliases:
- /resources/tools/mongodb-insert-many
---

## About

The `mongodb-insert-many` tool inserts **multiple new documents** into a
specified MongoDB collection in a single bulk operation. This is highly
efficient for adding large amounts of data at once.

This tool takes one required parameter named `data`. This `data` parameter must
be a string containing a **JSON array of document objects**. Upon successful
insertion, the tool returns a JSON array containing the unique `_id` of **each**
new document that was created.

This tool is compatible with the following source type:

* [`mongodb`](../../sources/mongodb.md)

---

## Example

Here is an example configuration for a tool that logs multiple events at once.

```yaml
tools:
  log_batch_events:
    kind: mongodb-insert-many
    source: my-mongo-source
    description: Inserts a batch of event logs into the database.
    database: logging
    collection: events
    canonical: true
```

An LLM would call this tool by providing an array of documents as a JSON string
in the `data` parameter, like this:
`tool_code: log_batch_events(data='[{"event": "login", "user": "user1"}, {"event": "click", "user": "user2"}, {"event": "logout", "user": "user1"}]')`

---

## Reference

| **field**   | **type** | **required** | **description**                                                                                    |
|:------------|:---------|:-------------|:---------------------------------------------------------------------------------------------------|
| type        | string   | true         | Must be `mongodb-insert-many`.                                                                     |
| source      | string   | true         | The name of the `mongodb` source to use.                                                           |
| description | string   | true         | A description of the tool that is passed to the LLM.                                               |
| database    | string   | true         | The name of the MongoDB database containing the collection.                                        |
| collection  | string   | true         | The name of the MongoDB collection into which the documents will be inserted.                      |
| canonical   | bool     | true         | Determines if the data string is parsed using MongoDB's Canonical or Relaxed Extended JSON format. |
