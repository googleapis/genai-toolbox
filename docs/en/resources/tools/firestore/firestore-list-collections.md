---
title: "firestore-list-collections"
type: docs
weight: 1
description: > 
  A "firestore-list-collections" tool lists collections in Firestore, either at the root level or as subcollections of a document.
aliases:
- /resources/tools/firestore-list-collections
---

## About

A `firestore-list-collections` tool lists collections in Firestore, either at the root level or as subcollections of a specific document.
It's compatible with the following sources:

- [firestore](../sources/firestore.md)

`firestore-list-collections` takes an optional `parentPath` parameter to specify a document
path. If provided, it lists all subcollections of that document. If not provided, it lists
all root-level collections in the database.

## Example

```yaml
tools:
  list_firestore_collections:
    kind: firestore-list-collections
    source: my-firestore-source
    description: Use this tool to list collections in Firestore.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "firestore-list-collections".                                                            |
| source      |                   string                   |     true     | Name of the Firestore source to list collections from.                                           |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
