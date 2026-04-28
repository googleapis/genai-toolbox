---
title: "cloud-storage-upload-object"
type: docs
weight: 6
description: >
  A "cloud-storage-upload-object" tool uploads a local file from the Toolbox server filesystem to a Cloud Storage object.
---

## About

A `cloud-storage-upload-object` tool streams a local file from the Toolbox
server filesystem into a Cloud Storage object. The `source` path is interpreted
on the server where Toolbox is running. Relative paths and paths containing `..`
are rejected.

When `content_type` is empty, Toolbox infers a MIME type from the source file
extension. If inference fails, Cloud Storage detects the content type from the
uploaded bytes.

[gcs-objects]: https://cloud.google.com/storage/docs/objects

## Compatible Sources

{{< compatible-sources >}}

## Requirements

The Cloud Storage credentials must be able to create or update the target
object. The Toolbox server process must also be able to read the local source
file.

## Parameters

| **parameter** | **type** | **required** | **description**                                                                                                   |
|---------------|:--------:|:------------:|-------------------------------------------------------------------------------------------------------------------|
| bucket        |  string  |     true     | Name of the Cloud Storage bucket to upload into.                                                                  |
| object        |  string  |     true     | Full object name (path) within the bucket, e.g. `path/to/file.txt`.                                               |
| source        |  string  |     true     | Absolute local filesystem path of the file to upload. Relative paths and paths containing `..` are rejected.       |
| content_type  |  string  |    false     | MIME type to record on the uploaded object. When empty, it is inferred from the source file extension when possible. |

## Example

```yaml
kind: tool
name: upload_object
type: cloud-storage-upload-object
source: my-gcs-source
description: Use this tool to upload a local file to Cloud Storage.
```

## Output Format

The tool returns a JSON object with:

| **field**   | **type** | **description**                              |
|-------------|:--------:|----------------------------------------------|
| bucket      |  string  | Cloud Storage bucket that received the file. |
| object      |  string  | Cloud Storage object name that was written.  |
| bytes       | integer  | Number of bytes uploaded.                    |
| contentType |  string  | Content type recorded on the uploaded object. |

## Reference

| **field**   | **type** | **required** | **description**                                        |
|-------------|:--------:|:------------:|--------------------------------------------------------|
| type        |  string  |     true     | Must be "cloud-storage-upload-object".                 |
| source      |  string  |     true     | Name of the Cloud Storage source to upload objects to. |
| description |  string  |     true     | Description of the tool that is passed to the LLM.     |
