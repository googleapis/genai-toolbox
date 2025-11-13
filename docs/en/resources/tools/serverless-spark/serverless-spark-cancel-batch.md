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

 `serverless-spark-cancel-batch` tool cancels a running Spark batch operation in
 a Google Cloud Serverless for Apache Spark source. The cancellation request is 
 asynchronous, so the batch state will not change immediately after the tool
 returns; it can take a minute or so for the cancellation to be reflected.

It's compatible with the following sources:

- [serverless-spark](../../sources/serverless-spark.md)

`serverless-spark-cancel-batch` accepts the following parameters:

- **`operation`** (required): The name of the operation to cancel. For example, for `projects/my-project/locations/us-central1/operations/my-operation`, you would pass `my-operation`.

The tool inherits the `project` and `location` from the source configuration.

## Example

```yaml
tools:
  cancel_spark_batch:
    kind: serverless-spark-cancel-batch
    source: my-serverless-spark-source
    description: Use this tool to cancel a running serverless spark batch operation.
```

## Response Format

```json
"Cancelled [projects/my-project/regions/us-central1/operations/my-operation]."
```

## Reference

| **field**    | **type** | **required** | **description**                                    |
| ------------ | :------: | :----------: | -------------------------------------------------- |
| kind         |  string  |     true     | Must be "serverless-spark-cancel-batch".           |
| source       |  string  |     true     | Name of the source the tool should use.            |
| description  |  string  |     true     | Description of the tool that is passed to the LLM. |
| authRequired | string[] |    false     | List of auth services required to invoke this tool |
