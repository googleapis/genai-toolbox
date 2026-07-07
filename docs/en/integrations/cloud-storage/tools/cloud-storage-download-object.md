---
title: "cloud-storage-download-object"
type: docs
weight: 5
description: >
  A "cloud-storage-download-object" tool downloads a Cloud Storage object to an absolute path on the Toolbox server filesystem.
---

## About

A `cloud-storage-download-object` tool streams a Cloud Storage object to a local
file on the Toolbox server. Unlike `cloud-storage-read-object`, it does not
return the object bytes to the LLM and does not require UTF-8 text content, so it
can be used for binary objects or large files.

The `destination` path is interpreted on the server where Toolbox is running.
By default, `destination` must be an absolute path and paths containing `..` are
rejected. When `destination_dir` is configured on the tool, `destination` remains
visible to the LLM but must be a relative path under that configured directory.
Absolute paths and paths that escape `destination_dir` are rejected.

[gcs-objects]: https://cloud.google.com/storage/docs/objects

## Compatible Sources

{{< compatible-sources >}}

## Requirements

The Cloud Storage credentials must be able to read the object. The Toolbox
server process must also be able to create or overwrite the destination file.
When `overwrite` is false, the tool returns an agent-fixable error if the
destination already exists.

## Parameters

| **parameter** | **type** | **required** | **description**                                                                                         |
|---------------|:--------:|:------------:|---------------------------------------------------------------------------------------------------------|
| bucket        |  string  |     true     | Name of the Cloud Storage bucket containing the object.                                                 |
| object        |  string  |     true     | Full object name (path) within the bucket, e.g. `path/to/file.txt`.                                     |
| destination   |  string  |     true     | Absolute local filesystem path where the object will be written. Relative paths and paths containing `..` are rejected. |
| overwrite     | boolean  |    false     | If true, overwrite the destination when it already exists. If false (default), return an error when it exists.     |

If `bucket` is configured on the tool, it is removed from the parameter list. If
`destination_dir` is configured on the tool, `destination` stays in the parameter
list but changes to a relative path under `destination_dir`. If `overwrite` is
configured on the tool, it is removed from the parameter list.

## Example

```yaml
kind: tool
name: download_object
type: cloud-storage-download-object
source: my-gcs-source
description: Use this tool to download a Cloud Storage object to the server filesystem.
```

```yaml
kind: tool
name: download_reports
type: cloud-storage-download-object
source: my-gcs-source
description: Use this tool to download report objects into the reports directory.
bucket: analytics-exports
destination_dir: /var/toolbox/downloads/reports
overwrite: false
```

## Output Format

The tool returns a JSON object with:

| **field**   | **type** | **description**                                |
|-------------|:--------:|------------------------------------------------|
| destination |  string  | Local path where the object was written.       |
| bytes       | integer  | Number of bytes written.                       |
| contentType |  string  | Content type recorded on the Cloud Storage object. |

## Reference

| **field**   | **type** | **required** | **description**                                           |
|-------------|:--------:|:------------:|-----------------------------------------------------------|
| type        |  string  |     true     | Must be "cloud-storage-download-object".                  |
| source      |  string  |     true     | Name of the Cloud Storage source to download objects from. |
| description |  string  |     true     | Description of the tool that is passed to the LLM.        |
| bucket      |  string  |    false     | Cloud Storage bucket to use for every invocation. When set, `bucket` is hidden from the tool parameters. |
| destination_dir | string | false | Absolute local directory to contain downloaded files. When set, `destination` is a relative path under this directory. |
| overwrite   | boolean  |    false     | Whether to overwrite existing destination files for every invocation. When set, `overwrite` is hidden from the tool parameters. |
