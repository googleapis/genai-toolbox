---
title: "cloud-storage-get-bucket-iam-policy"
type: docs
weight: 4
description: >
  A "cloud-storage-get-bucket-iam-policy" tool returns IAM policy bindings for a Cloud Storage bucket.
---

## About

A `cloud-storage-get-bucket-iam-policy` tool returns the IAM policy bindings for
a Cloud Storage bucket. Use it to inspect which principals have roles on a
bucket without modifying access.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

The Cloud Storage credentials must be able to read the IAM policy for the target
bucket.

## Parameters

| **parameter** | **type** | **required** | **description** |
|---------------|:--------:|:------------:|-----------------|
| bucket | string | true | Name of the Cloud Storage bucket whose IAM policy should be returned. |

## Example

```yaml
kind: tool
name: get_bucket_iam_policy
type: cloud-storage-get-bucket-iam-policy
source: my-gcs-source
description: Use this tool to inspect IAM bindings for a Cloud Storage bucket.
```

## Output Format

The tool returns a JSON object with:

| **field** | **type** | **description** |
|-----------|:--------:|-----------------|
| bucket | string | Cloud Storage bucket whose policy was read. |
| bindings | array | IAM bindings with `role`, `members`, and optional `condition` fields. |

## Reference

| **field** | **type** | **required** | **description** |
|-----------|:--------:|:------------:|-----------------|
| type | string | true | Must be "cloud-storage-get-bucket-iam-policy". |
| source | string | true | Name of the Cloud Storage source to get bucket IAM policies from. |
| description | string | true | Description of the tool that is passed to the LLM. |
