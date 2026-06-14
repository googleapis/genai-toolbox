---
title: "cloud-storage-list-buckets"
type: docs
weight: 1
description: >
  A "cloud-storage-list-buckets" tool lists Cloud Storage buckets in a project, with optional prefix filtering and pagination.
---

## About

A `cloud-storage-list-buckets` tool returns the Cloud Storage buckets in a
project. By default, it uses the project configured on the source. You can pass
the optional `project` parameter to list buckets in a different project that
the same credentials can access.

The response is a JSON object with `buckets` (bucket metadata returned by the
Cloud Storage API) and `nextPageToken` (empty when there are no more pages).

[gcs-buckets]: https://cloud.google.com/storage/docs/buckets

## Compatible Sources

{{< compatible-sources >}}

## Parameters

| **parameter** | **type** | **required** | **description**                                                                                                   |
|---------------|:--------:|:------------:|-------------------------------------------------------------------------------------------------------------------|
| project       |  string  |    false     | Project ID to list buckets in. When empty, the source's configured project is used.                               |
| prefix        |  string  |    false     | Filter results to buckets whose names begin with this prefix.                                                     |
| max_results   | integer  |    false     | Maximum number of buckets to return per page. A value of 0 uses the API default (1000); negative values and values above 1000 are rejected. |
| page_token    |  string  |    false     | A previously-returned page token for retrieving the next page of results.                                         |

## Example

```yaml
kind: tool
name: list_buckets
type: cloud-storage-list-buckets
source: my-gcs-source
description: Use this tool to list Cloud Storage buckets in the project.
```

## Output Format

The tool returns a JSON object with:

| **field**     | **type** | **description**                                  |
|---------------|:--------:|--------------------------------------------------|
| buckets       |  array   | Bucket metadata returned by the Cloud Storage API. |
| nextPageToken |  string  | Token to pass as `page_token` for the next page. |

## Reference

| **field**   | **type** | **required** | **description**                                         |
|-------------|:--------:|:------------:|---------------------------------------------------------|
| type        |  string  |     true     | Must be "cloud-storage-list-buckets".                   |
| source      |  string  |     true     | Name of the Cloud Storage source to list buckets from.  |
| description |  string  |     true     | Description of the tool that is passed to the LLM.      |
