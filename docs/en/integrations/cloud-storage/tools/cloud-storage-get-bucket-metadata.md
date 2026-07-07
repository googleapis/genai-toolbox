---
title: "cloud-storage-get-bucket-metadata"
type: docs
weight: 3
description: >
  A "cloud-storage-get-bucket-metadata" tool returns metadata for a Cloud Storage bucket.
---

## About

A `cloud-storage-get-bucket-metadata` tool returns metadata for a single Cloud
Storage bucket. Use it when the LLM needs fields such as location, storage
class, labels, lifecycle configuration, or uniform bucket-level access status.

The response is the bucket metadata structure returned by the Cloud Storage API.

You can set `bucket` in the tool configuration. When set, `bucket` is removed
from the runtime parameter schema and the configured bucket is always used. A
configured `bucket` must be a non-empty string.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

The Cloud Storage credentials must be able to read metadata for the target
bucket.

## Parameters

| **parameter** | **type** | **required** | **description** |
|---------------|:--------:|:------------:|-----------------|
| bucket | string | true | Name of the Cloud Storage bucket to inspect. |

## Example

```yaml
kind: tool
name: get_bucket_metadata
type: cloud-storage-get-bucket-metadata
source: my-gcs-source
description: Use this tool to inspect metadata for a Cloud Storage bucket.
```

```yaml
kind: tool
name: get_app_bucket_metadata
type: cloud-storage-get-bucket-metadata
source: my-gcs-source
description: Use this tool to inspect metadata for the application bucket.
bucket: my-app-bucket
```

## Output Format

The tool returns bucket metadata from the Cloud Storage API, including fields
such as `Name`, `Location`, `StorageClass`, `Created`, `Labels`,
`VersioningEnabled`, `Lifecycle`, and `UniformBucketLevelAccess`.

## Reference

| **field** | **type** | **required** | **description** |
|-----------|:--------:|:------------:|-----------------|
| type | string | true | Must be "cloud-storage-get-bucket-metadata". |
| source | string | true | Name of the Cloud Storage source to get bucket metadata from. |
| description | string | true | Description of the tool that is passed to the LLM. |
| bucket | string | false | Bucket to always inspect. When set, the runtime `bucket` parameter is hidden. Must not be empty. |
