---
title: "serverless-spark-get-session-template"
type: docs
weight: 1
description: >
  A "serverless-spark-get-session-template" tool retrieves a specific Spark session template from the source.
aliases:
  - /resources/tools/serverless-spark-get-session-template
---

## About

A `serverless-spark-get-session-template` tool retrieves a specific Spark session template from a
Google Cloud Serverless for Apache Spark source. It's compatible with the
following sources:

- [serverless-spark](../../sources/serverless-spark.md)

`serverless-spark-get-session-template` accepts the following parameters:

- **`name`** (required): The short name of the session template, e.g. for `projects/my-project/locations/us-central1/sessionTemplates/my-session-template`, pass `my-session-template`.

The tool gets the `project` and `location` from the source configuration.

## Example

```yaml
kind: tool
name: get_spark_session_template
type: serverless-spark-get-session-template
source: my-serverless-spark-source
description: Use this tool to get details of a serverless spark session template.
```

## Response Format

```json
{
  "sessionTemplate": {  
    "name": "projects/my-project/locations/us-central1/sessionTemplates/my-session-template",
    "description": "Template for Spark Session",
    // ... complete session template resource definition
  }
}
```

## Reference

| **field**    | **type** | **required** | **description**                                    |
| ------------ | :------: | :----------: | -------------------------------------------------- |
| type         |  string  |     true     | Must be "serverless-spark-get-session-template".   |
| source       |  string  |     true     | Name of the source the tool should use.            |
| description  |  string  |     true     | Description of the tool that is passed to the LLM. |
| authRequired | string[] |    false     | List of auth services required to invoke this tool |
  