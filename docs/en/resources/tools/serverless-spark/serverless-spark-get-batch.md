---
title: "serverless-spark-get-batch"
type: docs
weight: 1
description: >
  A "serverless-spark-get-batch" tool gets a single Spark batch from the source.
aliases:
  - /resources/tools/serverless-spark-get-batch
---

# serverless-spark-get-batch

The `serverless-spark-get-batch` tool allows you to retrieve a specific
Serverless Spark batch job. It's compatible with the following sources:

- [serverless-spark](../../sources/serverless-spark.md)

`serverless-spark-list-batches` accepts the following parameters:

- **`name`**: The short name of the batch, e.g. for
  `projects/my-project/locations/us-central1/my-batch`, pass `my-batch`.

The tool gets the `project` and `location` from the source configuration.

## Example

```yaml
tools:
  get_my_batch:
    kind: serverless-spark-get-batch
    source: my-serverless-spark-source
    description: Use this tool to get a serverless spark batch.
```

## Response Format

The response contains the full Batch object as defined in the [API
spec](https://cloud.google.com/dataproc-serverless/docs/reference/rest/v1/projects.locations.batches#Batch),
plus additional fields `consoleUrl` and `logsUrl` where a human can go for more
detailed information.

```json
{
  "batch": {
    "createTime": "2025-10-10T15:15:21.303146Z",
    "creator": "alice@example.com",
    "labels": {
      "goog-dataproc-batch-uuid": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
      "goog-dataproc-location": "us-central1"
    },
    "name": "projects/google.com:hadoop-cloud-dev/locations/us-central1/batches/alice-20251010-abcd",
    "operation": "projects/google.com:hadoop-cloud-dev/regions/us-central1/operations/11111111-2222-3333-4444-555555555555",
    "runtimeConfig": {
      "properties": {
        "spark:spark.driver.cores": "4",
        "spark:spark.driver.memory": "12200m"
      }
    },
    "sparkBatch": {
      "jarFileUris": [
        "file:///usr/lib/spark/examples/jars/spark-examples.jar"
      ],
      "mainClass": "org.apache.spark.examples.SparkPi"
    },
    "state": "SUCCEEDED",
    "stateHistory": [
      {
        "state": "PENDING",
        "stateStartTime": "2025-10-10T15:15:21.303146Z"
      },
      {
        "state": "RUNNING",
        "stateStartTime": "2025-10-10T15:16:41.291747Z"
      }
    ],
    "stateTime": "2025-10-10T15:17:21.265493Z",
    "uuid": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
  },
  "consoleUrl": "https://console.cloud.google.com/dataproc/batches/...",
  "logsUrl": "https://console.cloud.google.com/logs/viewer?..."
}
```

## Reference

| **field**    | **type** | **required** | **description**                                    |
| ------------ | :------: | :----------: | -------------------------------------------------- |
| kind         |  string  |     true     | Must be "serverless-spark-get-batch".              |
| source       |  string  |     true     | Name of the source the tool should use.            |
| description  |  string  |     true     | Description of the tool that is passed to the LLM. |
| authRequired | string[] |    false     | List of auth services required to invoke this tool |
