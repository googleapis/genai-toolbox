---
title: "cloud-storage-copy-object"
type: docs
weight: 8
description: >
  A "cloud-storage-copy-object" tool copies a Cloud Storage object to another object, including across buckets.
---

## About

A `cloud-storage-copy-object` tool copies an object from one Cloud Storage
location to another. The source and destination bucket parameters are separate,
so the destination can be in the same bucket or a different bucket.

Existing destination objects are replaced.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

The Cloud Storage credentials must be able to read the source object and create
or update the destination object.

## Parameters

| **parameter**      | **type** | **required** | **description**                                                                    |
|--------------------|:--------:|:------------:|------------------------------------------------------------------------------------|
| source_bucket      |  string  |     true     | Name of the Cloud Storage bucket containing the source object.                      |
| source_object      |  string  |     true     | Full source object name (path) within the source bucket, e.g. `path/to/file.txt`.   |
| destination_bucket |  string  |     true     | Name of the Cloud Storage bucket to copy into.                                      |
| destination_object |  string  |     true     | Full destination object name (path) within the destination bucket.                  |

## Example

```yaml
kind: tool
name: copy_object
type: cloud-storage-copy-object
source: my-gcs-source
description: Use this tool to copy Cloud Storage objects.
```

## Output Format

The tool returns a JSON object with:

| **field**          | **type** | **description**                                  |
|--------------------|:--------:|--------------------------------------------------|
| sourceBucket       |  string  | Source Cloud Storage bucket.                     |
| sourceObject       |  string  | Source Cloud Storage object name.                |
| destinationBucket  |  string  | Destination Cloud Storage bucket.                |
| destinationObject  |  string  | Destination Cloud Storage object name.           |
| bytes              | integer  | Size of the copied object.                       |
| contentType        |  string  | Content type recorded on the destination object. |

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| type        |  string  |     true     | Must be "cloud-storage-copy-object".                 |
| source      |  string  |     true     | Name of the Cloud Storage source to copy objects in. |
| description |  string  |     true     | Description of the tool that is passed to the LLM.   |
