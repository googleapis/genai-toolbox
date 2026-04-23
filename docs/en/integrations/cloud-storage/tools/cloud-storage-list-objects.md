---
title: "cloud-storage-list-objects"
type: docs
weight: 1
description: >
  A "cloud-storage-list-objects" tool lists objects in a Cloud Storage bucket, with optional prefix filtering and delimiter-based grouping.
---

## About

A `cloud-storage-list-objects` tool returns the objects in a
[Cloud Storage bucket][gcs-buckets]. It supports the usual GCS listing options:

- `prefix` — filter results to objects whose names begin with the given string.
- `delimiter` — group results by this character (typically `/`) so subdirectory-like
  "common prefixes" are returned separately from the leaf objects.
- `max_results` / `page_token` — paginate through large listings.

The response is a JSON object with `objects` (the full object metadata as
returned by the Cloud Storage API — fields such as `Name`, `Size`, `ContentType`,
`Updated`, `StorageClass`, `MD5`, etc.), `prefixes` (the common prefixes when
`delimiter` is set), and `nextPageToken` (empty when there are no more pages).

[gcs-buckets]: https://cloud.google.com/storage/docs/buckets

## Compatible Sources

{{< compatible-sources >}}

## Parameters

| **parameter** | **type** | **required** | **description**                                                                                                   |
|---------------|:--------:|:------------:|-------------------------------------------------------------------------------------------------------------------|
| bucket        |  string  |     true     | Name of the Cloud Storage bucket to list objects from.                                                            |
| prefix        |  string  |    false     | Filter results to objects whose names begin with this prefix.                                                     |
| delimiter     |  string  |    false     | Delimiter used to group object names (typically '/'). When set, common prefixes are returned as `prefixes`.       |
| max_results   | integer  |    false     | Maximum number of objects to return per page. A value of 0 uses the API default (1000); values above 1000 are rejected. |
| page_token    |  string  |    false     | A previously-returned page token for retrieving the next page of results.                                         |

## Example

```yaml
kind: tool
name: list_objects
type: cloud-storage-list-objects
source: my-gcs-source
description: Use this tool to list objects in a Cloud Storage bucket.
```

## Reference

| **field**   | **type** | **required** | **description**                                         |
|-------------|:--------:|:------------:|---------------------------------------------------------|
| type        |  string  |     true     | Must be "cloud-storage-list-objects".                   |
| source      |  string  |     true     | Name of the Cloud Storage source to list objects from.  |
| description |  string  |     true     | Description of the tool that is passed to the LLM.      |
