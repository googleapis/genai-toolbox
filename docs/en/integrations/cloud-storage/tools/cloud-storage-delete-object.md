---
title: "cloud-storage-delete-object"
type: docs
weight: 10
description: >
  A "cloud-storage-delete-object" tool deletes a Cloud Storage object.
---

## About

A `cloud-storage-delete-object` tool deletes a single object from a Cloud
Storage bucket.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

The Cloud Storage credentials must be able to delete the target object.

## Parameters

| **parameter** | **type** | **required** | **description**                                                     |
|---------------|:--------:|:------------:|---------------------------------------------------------------------|
| bucket        |  string  |     true     | Name of the Cloud Storage bucket containing the object to delete.    |
| object        |  string  |     true     | Full object name (path) within the bucket, e.g. `path/to/file.txt`.  |

## Example

```yaml
kind: tool
name: delete_object
type: cloud-storage-delete-object
source: my-gcs-source
description: Use this tool to delete Cloud Storage objects.
```

## Output Format

The tool returns a JSON object with:

| **field** | **type** | **description**                             |
|-----------|:--------:|---------------------------------------------|
| bucket    |  string  | Cloud Storage bucket containing the object. |
| object    |  string  | Cloud Storage object name that was deleted. |
| deleted   | boolean  | Whether the delete request completed.       |

## Reference

| **field**   | **type** | **required** | **description**                                        |
|-------------|:--------:|:------------:|--------------------------------------------------------|
| type        |  string  |     true     | Must be "cloud-storage-delete-object".                 |
| source      |  string  |     true     | Name of the Cloud Storage source to delete objects in. |
| description |  string  |     true     | Description of the tool that is passed to the LLM.     |
