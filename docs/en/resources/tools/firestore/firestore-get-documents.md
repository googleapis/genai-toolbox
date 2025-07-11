---
title: "firestore-get-documents"
type: docs
weight: 1
description: > 
  A "firestore-get-documents" tool retrieves multiple documents from Firestore by their paths.
aliases:
- /resources/tools/firestore-get-documents
---

## About

A `firestore-get-documents` tool retrieves multiple documents from Firestore by their paths.
It's compatible with the following sources:

- [firestore](../sources/firestore.md)

`firestore-get-documents` takes one input parameter `documentPaths` which is an array of 
document paths, and returns the documents' data along with metadata such as existence status,
creation time, update time, and read time.

## Example

```yaml
tools:
  get_user_documents:
    kind: firestore-get-documents
    source: my-firestore-source
    description: Use this tool to retrieve multiple documents from Firestore.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "firestore-get-documents".                                                               |
| source      |                   string                   |     true     | Name of the Firestore source to retrieve documents from.                                         |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
