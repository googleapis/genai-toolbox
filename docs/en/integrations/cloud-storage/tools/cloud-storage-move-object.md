---
title: "cloud-storage-move-object"
type: docs
weight: 9
description: >
  A "cloud-storage-move-object" tool atomically moves or renames a Cloud Storage object within the same bucket.
---

## About

A `cloud-storage-move-object` tool atomically moves or renames an object within
the same Cloud Storage bucket by using Cloud Storage's native move API.

This tool does not perform cross-bucket moves. For a cross-bucket move, call
`cloud-storage-copy-object` first, verify the destination, and then call
`cloud-storage-delete-object` on the source.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

The Cloud Storage credentials must have `storage.objects.move` and
`storage.objects.create` permissions in the bucket. If the destination object
already exists, `storage.objects.delete` is also required.

## Parameters

| **parameter**      | **type** | **required** | **description**                                                                 |
|--------------------|:--------:|:------------:|---------------------------------------------------------------------------------|
| bucket             |  string  |     true     | Name of the Cloud Storage bucket containing the object to move.                  |
| source_object      |  string  |     true     | Full source object name (path) within the bucket, e.g. `path/to/file.txt`.       |
| destination_object |  string  |     true     | Full destination object name (path) within the same bucket.                      |

## Example

```yaml
kind: tool
name: move_object
type: cloud-storage-move-object
source: my-gcs-source
description: Use this tool to move or rename an object within a Cloud Storage bucket.
```

## Output Format

The tool returns a JSON object with:

| **field**          | **type** | **description**                                |
|--------------------|:--------:|------------------------------------------------|
| bucket             |  string  | Cloud Storage bucket containing the object.    |
| sourceObject       |  string  | Original object name.                          |
| destinationObject  |  string  | Destination object name.                       |
| bytes              | integer  | Size of the moved object.                      |
| contentType        |  string  | Content type recorded on the destination object. |

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| type        |  string  |     true     | Must be "cloud-storage-move-object".                 |
| source      |  string  |     true     | Name of the Cloud Storage source to move objects in. |
| description |  string  |     true     | Description of the tool that is passed to the LLM.   |
