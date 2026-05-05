---
title: "cloud-storage-create-bucket"
type: docs
weight: 2
description: >
  A "cloud-storage-create-bucket" tool creates a Cloud Storage bucket in the configured source project.
---

## About

A `cloud-storage-create-bucket` tool creates a new Cloud Storage bucket in the
project configured on the Cloud Storage source. Use it when an agent needs to
provision a bucket before writing objects or building a data workflow.

[gcs-buckets]: https://cloud.google.com/storage/docs/buckets

## Compatible Sources

{{< compatible-sources >}}

## Requirements

The Cloud Storage credentials must be able to create buckets in the configured
project. Bucket names are globally unique and must satisfy Cloud Storage bucket
naming rules.

## Parameters

| **parameter** | **type** | **required** | **description** |
|---------------|:--------:|:------------:|-----------------|
| bucket | string | true | Name of the Cloud Storage bucket to create. |
| location | string | false | Location for the bucket, e.g. "US", "EU", or "us-central1". Omit to use the Cloud Storage service default. |
| uniform_bucket_level_access | boolean | false | Whether to enable uniform bucket-level access on the bucket. Defaults to false. |

## Example

```yaml
kind: tool
name: create_bucket
type: cloud-storage-create-bucket
source: my-gcs-source
description: Use this tool to create Cloud Storage buckets.
```

## Output Format

The tool returns a JSON object with:

| **field** | **type** | **description** |
|-----------|:--------:|-----------------|
| bucket | string | Cloud Storage bucket that was created. |
| created | boolean | Whether the bucket was created. |
| metadata | object | Bucket metadata returned by the Cloud Storage API. |

## Reference

| **field** | **type** | **required** | **description** |
|-----------|:--------:|:------------:|-----------------|
| type | string | true | Must be "cloud-storage-create-bucket". |
| source | string | true | Name of the Cloud Storage source to create buckets from. |
| description | string | true | Description of the tool that is passed to the LLM. |
