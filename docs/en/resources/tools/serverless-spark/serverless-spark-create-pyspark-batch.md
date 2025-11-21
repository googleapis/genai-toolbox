---
title: "serverless-spark-create-pyspark-batch"
type: docs
weight: 2
description: >
  A "serverless-spark-create-pyspark-batch" tool submits a Spark batch to run asynchronously.
aliases:
  - /resources/tools/serverless-spark-create-pyspark-batch
---

## About

A `serverless-spark-create-pyspark-batch` tool submits a Spark batch to a Google
Cloud Serverless for Apache Spark source. The workload executes asynchronously
and takes around a minute to begin executing; status can be polled using the
[get batch](serverless-spark-get-batch.md) tool.

It's compatible with the following sources:

- [serverless-spark](../../sources/serverless-spark.md)

`serverless-spark-create-pyspark-batch` accepts the following parameters:

- **`mainFile`**: The path to the main Python file, as a gs://... URI.
- **`args`** Optional. A list of arguments passed to the main file.
- **`version`** Optional. The Serverless [runtime
  version](https://docs.cloud.google.com/dataproc-serverless/docs/concepts/versions/dataproc-serverless-versions)
  to execute with.

## Custom Configuration

This tool supports custom
[`runtimeConfig`](https://docs.cloud.google.com/dataproc-serverless/docs/reference/rest/v1/RuntimeConfig)
and
[`environmentConfig`](https://docs.cloud.google.com/dataproc-serverless/docs/reference/rest/v1/EnvironmentConfig)
settings, which can be specified in a `tools.yaml` file. These configurations
are parsed as YAML and passed to the Dataproc API.

**Note:** If your project requires custom runtime or environment configuration,
you must write a custom `tools.yaml`, you cannot use the `serverless-spark`
prebuilt config.

### Example `tools.yaml`

```yaml
tools:
  - name: "serverless-spark-create-pyspark-batch"
    kind: "serverless-spark-create-pyspark-batch"
    source: "my-serverless-spark-source"
    runtimeConfig:
      properties:
        spark.driver.memory: "1024m"
    environmentConfig:
      executionConfig:
        networkUri: "my-network"
```

## Response Format

The response contains the
[operation](https://docs.cloud.google.com/dataproc-serverless/docs/reference/rest/v1/projects.locations.operations#resource:-operation)
metadata JSON object corresponding to [batch operation
metadata](https://pkg.go.dev/cloud.google.com/go/dataproc/v2/apiv1/dataprocpb#BatchOperationMetadata),
plus additional fields `consoleUrl` and `logsUrl` where a human can go for more
detailed information.

```json
{
  "opMetadata": {
    "batch": "projects/myproject/locations/us-central1/batches/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
    "batchUuid": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
    "createTime": "2025-11-19T16:36:47.607119Z",
    "description": "Batch",
    "labels": {
      "goog-dataproc-batch-uuid": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
      "goog-dataproc-location": "us-central1"
    },
    "operationType": "BATCH",
    "warnings": [
      "No runtime version specified. Using the default runtime version."
    ]
  },
  "consoleUrl": "https://console.cloud.google.com/dataproc/batches/...",
  "logsUrl": "https://console.cloud.google.com/logs/viewer?..."
}
```

## Reference

| **field**         | **type** | **required** | **description**                                                                                                                                          |
| ----------------- | :------: | :----------: | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| kind              |  string  |     true     | Must be "serverless-spark-create-pyspark-batch".                                                                                                         |
| source            |  string  |     true     | Name of the source the tool should use.                                                                                                                  |
| description       |  string  |    false     | Description of the tool that is passed to the LLM.                                                                                                       |
| runtimeConfig     |   map    |    false     | [Runtime config](https://docs.cloud.google.com/dataproc-serverless/docs/reference/rest/v1/RuntimeConfig) for all batches created with this tool.         |
| environmentConfig |   map    |    false     | [Environment config](https://docs.cloud.google.com/dataproc-serverless/docs/reference/rest/v1/EnvironmentConfig) for all batches created with this tool. |
| authRequired      | string[] |    false     | List of auth services required to invoke this tool.                                                                                                      |
