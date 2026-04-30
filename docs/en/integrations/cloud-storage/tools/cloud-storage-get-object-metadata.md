---
title: "cloud-storage-get-object-metadata"
type: docs
weight: 3
description: >
  A "cloud-storage-get-object-metadata" tool returns metadata for a Cloud Storage object without reading the object payload.
---

## About

A `cloud-storage-get-object-metadata` tool returns metadata for a single
[Cloud Storage object][gcs-objects]. Use it when the LLM needs fields such as
object name, size, content type, generation, storage class, timestamps, checksums,
or custom metadata without reading the object's content.

The response is the object metadata structure returned by the Cloud Storage API.

[gcs-objects]: https://cloud.google.com/storage/docs/objects

## Compatible Sources

{{< compatible-sources >}}

## Parameters

| **parameter** | **type** | **required** | **description**                                                     |
|---------------|:--------:|:------------:|---------------------------------------------------------------------|
| bucket        |  string  |     true     | Name of the Cloud Storage bucket containing the object.             |
| object        |  string  |     true     | Full object name (path) within the bucket, e.g. `path/to/file.txt`. |

## Example

```yaml
kind: tool
name: get_object_metadata
type: cloud-storage-get-object-metadata
source: my-gcs-source
description: Use this tool to inspect metadata for a Cloud Storage object.
```

## Output Format

The tool returns object metadata from the Cloud Storage API, including fields
such as `Name`, `Bucket`, `Size`, `ContentType`, `Updated`, `StorageClass`,
`MD5`, `CRC32C`, and user-defined metadata when present.

## Reference

| **field**   | **type** | **required** | **description**                                             |
|-------------|:--------:|:------------:|-------------------------------------------------------------|
| type        |  string  |     true     | Must be "cloud-storage-get-object-metadata".                |
| source      |  string  |     true     | Name of the Cloud Storage source to get object metadata from. |
| description |  string  |     true     | Description of the tool that is passed to the LLM.          |
