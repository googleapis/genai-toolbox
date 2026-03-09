---
title: "serverless-spark-get-session"
type: docs
weight: 1
description: >
  A "serverless-spark-get-session" tool retrieves a specific Spark session from the source.
aliases:
  - /resources/tools/serverless-spark-get-session
---

## About

A `serverless-spark-get-session` tool retrieves a specific Spark session from a
Google Cloud Serverless for Apache Spark source. It's compatible with the
following sources:

- [serverless-spark](../../sources/serverless-spark.md)

`serverless-spark-get-session` accepts the following parameters:

- **`name`** (required): The short name of the session, e.g. for `projects/my-project/locations/us-central1/sessions/my-session`, pass `my-session`.

The tool gets the `project` and `location` from the source configuration.

## Example

```yaml
kind: tools
name: get_spark_session
type: serverless-spark-get-session
source: my-serverless-spark-source
description: Use this tool to get details of a serverless spark session.
```

## Response Format

```json
{
  "consoleUrl": "https://console.cloud.google.com/dataproc/interactive/us-central1/my-session/details?project=my-project",
  "logsUrl": "https://console.cloud.google.com/logs/viewer?...",
  "session": {  
    "name": "projects/my-project/locations/us-central1/sessions/my-session",
    "uuid": "a1b2c3d4-e5f6-7890-1234-567890abcdef",
    "state": "ACTIVE",
    // ... complete session resource definition
  }
}
```

## Reference

| **field**    | **type** | **required** | **description**                                    |
| ------------ | :------: | :----------: | -------------------------------------------------- |
| type         |  string  |     true     | Must be "serverless-spark-get-session".            |
| source       |  string  |     true     | Name of the source the tool should use.            |
| description  |  string  |     true     | Description of the tool that is passed to the LLM. |
| authRequired | string[] |    false     | List of auth services required to invoke this tool |
  