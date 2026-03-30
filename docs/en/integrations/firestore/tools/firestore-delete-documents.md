---
title: "firestore-delete-documents"
type: docs
weight: 1
description: >
  A "firestore-delete-documents" tool deletes multiple documents from Firestore by their paths.
---

## About

A `firestore-delete-documents` tool deletes multiple documents from Firestore by
their paths.

`firestore-delete-documents` takes one input parameter `documentPaths` which is
an array of document paths to delete. The tool uses Firestore's BulkWriter for
efficient batch deletion and returns the success status for each document.

## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: delete_user_documents
type: firestore-delete-documents
source: my-firestore-source
description: Use this tool to delete multiple documents from Firestore.
```

## Reference

| **field**   |     **type**   | **required** | **description**                                          |
|-------------|:--------------:|:------------:|----------------------------------------------------------|
| type        |     string     |     true     | Must be "firestore-delete-documents".                    |
| source      |     string     |     true     | Name of the Firestore source to delete documents from.   |
| description |     string     |     true     | Description of the tool that is passed to the LLM.       |
