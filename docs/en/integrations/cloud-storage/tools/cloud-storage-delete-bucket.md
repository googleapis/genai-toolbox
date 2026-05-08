---
title: "cloud-storage-delete-bucket"
type: docs
weight: 5
description: >
  A "cloud-storage-delete-bucket" tool deletes an empty Cloud Storage bucket.
---

## About

A `cloud-storage-delete-bucket` tool deletes an empty Cloud Storage bucket. It
does not delete objects first; if the bucket is not empty, Cloud Storage rejects
the operation.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

The Cloud Storage credentials must be able to delete the target bucket. The
bucket must be empty before the tool is invoked.

## Parameters

| **parameter** | **type** | **required** | **description** |
|---------------|:--------:|:------------:|-----------------|
| bucket | string | true | Name of the empty Cloud Storage bucket to delete. |

## Example

```yaml
kind: tool
name: delete_bucket
type: cloud-storage-delete-bucket
source: my-gcs-source
description: Use this tool to delete empty Cloud Storage buckets.
```

## Output Format

The tool returns a JSON object with:

| **field** | **type** | **description** |
|-----------|:--------:|-----------------|
| bucket | string | Cloud Storage bucket that was deleted. |
| deleted | boolean | Whether the bucket was deleted. |

## Reference

| **field** | **type** | **required** | **description** |
|-----------|:--------:|:------------:|-----------------|
| type | string | true | Must be "cloud-storage-delete-bucket". |
| source | string | true | Name of the Cloud Storage source to delete buckets from. |
| description | string | true | Description of the tool that is passed to the LLM. |
