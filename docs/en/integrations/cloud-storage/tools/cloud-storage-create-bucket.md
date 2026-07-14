---
title: "cloud-storage-create-bucket"
type: docs
weight: 2
description: >
  A "cloud-storage-create-bucket" tool creates a Cloud Storage bucket in a configured or runtime-selected project.
---

## About

A `cloud-storage-create-bucket` tool creates a new Cloud Storage bucket. By
default, it creates the bucket in the project configured on the Cloud Storage
source. You can pass the optional `project` parameter to create buckets in a
different project that the same credentials can access.

You can also set `project`, `location`, or `uniform_bucket_level_access` in the
tool configuration. When set, that field is removed from the runtime parameter
schema and the configured value is always used. Set `project` to an empty string
to hide the parameter while using the source's configured project.

[gcs-buckets]: https://cloud.google.com/storage/docs/buckets

## Compatible Sources

{{< compatible-sources >}}

## Requirements

The Cloud Storage credentials must be able to create buckets in the target
project. Bucket names are globally unique and must satisfy Cloud Storage bucket
naming rules.

## Parameters

| **parameter** | **type** | **required** | **description** |
|---------------|:--------:|:------------:|-----------------|
| bucket | string | true | Name of the Cloud Storage bucket to create. |
| project | string | false | Project ID to create the bucket in. When empty, the source's configured project is used. |
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

```yaml
kind: tool
name: create_us_bucket
type: cloud-storage-create-bucket
source: my-gcs-source
description: Use this tool to create Cloud Storage buckets in the US location.
project: ""
location: US
uniform_bucket_level_access: true
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
| project | string | false | Project ID to always use. When set, the runtime `project` parameter is hidden. An empty string uses the source's configured project. |
| location | string | false | Bucket location to always use. When set, the runtime `location` parameter is hidden. |
| uniform_bucket_level_access | boolean | false | Uniform bucket-level access setting to always use. When set, the runtime `uniform_bucket_level_access` parameter is hidden. |
