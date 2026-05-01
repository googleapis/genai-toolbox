---
title: "cloud-storage-write-object"
type: docs
weight: 7
description: >
  A "cloud-storage-write-object" tool writes text content directly to a Cloud Storage object.
---

## About

A `cloud-storage-write-object` tool writes text content from the tool request
directly into a Cloud Storage object. It is useful for creating or replacing
small text objects without first writing a local file on the Toolbox server.

When `content_type` is empty, Cloud Storage detects the content type from the
written bytes.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

The Cloud Storage credentials must be able to create or update the target
object.

## Parameters

| **parameter** | **type** | **required** | **description**                                                                                       |
|---------------|:--------:|:------------:|-------------------------------------------------------------------------------------------------------|
| bucket        |  string  |     true     | Name of the Cloud Storage bucket to write into.                                                       |
| object        |  string  |     true     | Full object name (path) within the bucket, e.g. `path/to/file.txt`.                                   |
| content       |  string  |     true     | Text content to write to the Cloud Storage object.                                                    |
| content_type  |  string  |    false     | MIME type to record on the written object. When empty, Cloud Storage auto-detects from the content.   |

## Example

```yaml
kind: tool
name: write_object
type: cloud-storage-write-object
source: my-gcs-source
description: Use this tool to write text content to Cloud Storage.
```

## Output Format

The tool returns a JSON object with:

| **field**   | **type** | **description**                              |
|-------------|:--------:|----------------------------------------------|
| bucket      |  string  | Cloud Storage bucket that received content.  |
| object      |  string  | Cloud Storage object name that was written.  |
| bytes       | integer  | Number of bytes written.                     |
| contentType |  string  | Content type recorded on the written object. |

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| type        |  string  |     true     | Must be "cloud-storage-write-object".                |
| source      |  string  |     true     | Name of the Cloud Storage source to write objects to. |
| description |  string  |     true     | Description of the tool that is passed to the LLM.   |
