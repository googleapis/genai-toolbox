---
title: "serverless-spark-list-batches"
type: docs
weight: 1
description: >
  A "serverless-spark-list-batches" tool returns a list of Spark batches from the source.
aliases:
  - /resources/tools/serverless-spark-list-batches
---

## About

A `serverless-spark-list-batches` tool returns a list of Spark batches from a
Google Cloud Serverless for Apache Spark source. It's compatible with the
following sources:

- [serverless-spark](../../sources/serverless-spark.md)

`serverless-spark-list-batches` accepts the following parameters:

- **`filter`** (optional): A filter expression to limit the batches returned.
  Filters are case sensitive and may contain multiple clauses combined with
  logical operators (AND/OR). Supported fields are `batch_id`, `batch_uuid`,
  `state`, `create_time`, and `labels`. For example: `state = RUNNING AND
create_time < "2023-01-01T00:00:00Z"`.
- **`pageSize`** (optional): The maximum number of batches to return in a single
  page.
- **`pageToken`** (optional): A page token, received from a previous call, to
  retrieve the next page of results.

The tool gets the `project` and `location` from the source configuration.

## Example

```yaml
tools:
  list_spark_batches:
    kind: serverless-spark-list-batches
    source: my-serverless-spark-source
    description: Use this tool to list and filter serverless spark batches.
```

## Response Format

```json
{
  "batches": [
    {
      "name": "projects/my-project/locations/us-central1/batches/batch-abc-123",
      "uuid": "a1b2c3d4-e5f6-7890-1234-567890abcdef",
      "state": "SUCCEEDED",
      "creator": "alice@example.com",
      "createTime": "2023-10-27T10:00:00Z",
      "consoleUrl": "https://console.cloud.google.com/dataproc/batches/us-central1/batch-abc-123/summary?project=my-project",
      "logsUrl": "https://console.cloud.google.com/logs/viewer?advancedFilter=resource.type%3D%22cloud_dataproc_batch%22%0Aresource.labels.project_id%3D%22my-project%22%0Aresource.labels.location%3D%22us-central1%22%0Aresource.labels.batch_id%3D%22batch-abc-123%22%0Atimestamp%3E%3D%222023-10-27T09%3A59%3A00Z%22%0Atimestamp%3C%3D%222023-10-27T10%3A10%3A00Z%22&project=my-project&resource=cloud_dataproc_batch%2Fbatch_id%2Fbatch-abc-123"
    },
    {
      "name": "projects/my-project/locations/us-central1/batches/batch-def-456",
      "uuid": "b2c3d4e5-f6a7-8901-2345-678901bcdefa",
      "state": "FAILED",
      "creator": "alice@example.com",
      "createTime": "2023-10-27T11:30:00Z",
      "consoleUrl": "https://console.cloud.google.com/dataproc/batches/us-central1/batch-def-456/summary?project=my-project",
      "logsUrl": "https://console.cloud.google.com/logs/viewer?advancedFilter=resource.type%3D%22cloud_dataproc_batch%22%0Aresource.labels.project_id%3D%22my-project%22%0Aresource.labels.location%3D%22us-central1%22%0Aresource.labels.batch_id%3D%22batch-def-456%22%0Atimestamp%3E%3D%222023-10-27T11%3A29%3A00Z%22%0Atimestamp%3C%3D%222023-10-27T11%3A40%3A00Z%22&project=my-project&resource=cloud_dataproc_batch%2Fbatch_id%2Fbatch-def-456"
    }
  ],
  "nextPageToken": "abcd1234"
}
```

## Reference

| **field**    | **type** | **required** | **description**                                    |
| ------------ | :------: | :----------: | -------------------------------------------------- |
| kind         |  string  |     true     | Must be "serverless-spark-list-batches".           |
| source       |  string  |     true     | Name of the source the tool should use.            |
| description  |  string  |     true     | Description of the tool that is passed to the LLM. |
| authRequired | string[] |    false     | List of auth services required to invoke this tool |
