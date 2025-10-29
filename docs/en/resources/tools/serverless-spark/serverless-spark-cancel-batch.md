---
title: "serverless-spark-cancel-batch"
type: docs
weight: 2
description: >
  A "serverless-spark-cancel-batch" tool cancels a running Spark batch operation.
aliases:
  - /resources/tools/serverless-spark-cancel-batch
---

## About

A `serverless-spark-cancel-batch` tool cancels a running Spark batch operation in a
Google Cloud Serverless for Apache Spark source. It's compatible with the
following sources:

- [serverless-spark](../../sources/serverless-spark.md)

`serverless-spark-cancel-batch` accepts the following parameters:

- **`operation`** (required): The name of the operation to cancel. For example, for `projects/my-project/locations/us-central1/operations/my-operation`, you would pass `my-operation`.

The tool gets the `project` and `location` from the source configuration.

## Example

```yaml
tools:
  cancel_spark_batch:
    kind: serverless-spark-cancel-batch
    source: my-serverless-spark-source
    description: Use this tool to cancel a running serverless spark batch operation.
```

## Response Format

The tool returns a string indicating the result of the cancellation.

```json
{
  "result": "Operation canceled successfully."
}
```

## Reference

| **field**    | **type** | **required** | **description**                                    |
| ------------ | :------: | :----------: | -------------------------------------------------- |
| kind         |  string  |     true     | Must be "serverless-spark-cancel-batch".           |
| source       |  string  |     true     | Name of the source the tool should use.            |
| description  |  string  |     true     | Description of the tool that is passed to the LLM. |
| authRequired | string[] |    false     | List of auth services required to invoke this tool |
