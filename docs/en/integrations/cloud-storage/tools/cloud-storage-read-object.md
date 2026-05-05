---
title: "cloud-storage-read-object"
type: docs
weight: 4
description: >
  A "cloud-storage-read-object" tool reads the UTF-8 text content of a Cloud Storage object, optionally constrained to a byte range.
---

## About

A `cloud-storage-read-object` tool fetches the bytes of a single
[Cloud Storage object][gcs-objects] and returns them as plain UTF-8 text.

Only text objects are supported today: if the object bytes (or the requested
range) are not valid UTF-8 the tool returns an agent-fixable error. This is
because the MCP tool-result channel currently only carries text; binary
payloads will be supported once MCP can carry embedded resources.

Reads are capped at **8 MiB** per call to protect the server's memory and keep
LLM contexts manageable; objects or ranges larger than that are rejected with
an agent-fixable error. Use the optional `range` parameter to read a slice of
a larger object.

This tool is intended for small-to-medium textual content an LLM can process
directly. For bulk downloads of large files to the local filesystem, use
`cloud-storage-download-object`.

[gcs-objects]: https://cloud.google.com/storage/docs/objects

## Compatible Sources

{{< compatible-sources >}}

## Parameters

| **parameter** | **type** | **required** | **description**                                                                                                                                   |
|---------------|:--------:|:------------:|---------------------------------------------------------------------------------------------------------------------------------------------------|
| bucket        |  string  |     true     | Name of the Cloud Storage bucket containing the object.                                                                                           |
| object        |  string  |     true     | Full object name (path) within the bucket, e.g. `path/to/file.txt`.                                                                               |
| range         |  string  |    false     | Optional HTTP byte range, e.g. `bytes=0-999` (first 1000 bytes), `bytes=-500` (last 500 bytes), or `bytes=500-` (from byte 500 to end). Empty reads the full object. |

## Example

```yaml
kind: tool
name: read_object
type: cloud-storage-read-object
source: my-gcs-source
description: Use this tool to read the content of a Cloud Storage object.
```

## Reference

| **field**   | **type** | **required** | **description**                                         |
|-------------|:--------:|:------------:|---------------------------------------------------------|
| type        |  string  |     true     | Must be "cloud-storage-read-object".                    |
| source      |  string  |     true     | Name of the Cloud Storage source to read the object from. |
| description |  string  |     true     | Description of the tool that is passed to the LLM.      |
