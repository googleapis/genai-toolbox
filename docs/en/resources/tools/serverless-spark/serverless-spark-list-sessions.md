---
title: "serverless-spark-list-sessions"
type: docs
weight: 1
description: >
  A "serverless-spark-list-sessions" tool returns a list of Spark sessions from the source.
aliases:
  - /resources/tools/serverless-spark-list-sessions
---

## About

A `serverless-spark-list-sessions` tool returns a list of Spark sessions from a
Google Cloud Serverless for Apache Spark source. It's compatible with the
following sources:

- [serverless-spark](../../sources/serverless-spark.md)

`serverless-spark-list-sessions` accepts the following parameters:

- **`filter`** (optional): Optional. A filter for the sessions to return in the
  response. A filter is a logical expression constraining the values of various
  fields in each session resource. Filters are case sensitive, and may contain
  multiple clauses combined with logical operators (AND, OR). Supported fields
  are session_id, session_uuid, state, create_time, and labels. Example: `state
  = ACTIVE and create_time < "2023-01-01T00:00:00Z"` is a filter for sessions in
  an ACTIVE state that were created before 2023-01-01. `state = ACTIVE and
  labels.environment=production` is a filter for sessions in an ACTIVE state
  that have a production environment label.
- **`pageSize`** (optional): The maximum number of sessions to return in a single
  page. Defaults to `20`.
- **`pageToken`** (optional): A page token, received from a previous call, to
  retrieve the next page of results.

The tool gets the `project` and `location` from the source configuration.

## Example

```yaml
kind: tool
name: list_spark_sessions
type: serverless-spark-list-sessions
source: my-serverless-spark-source
description: Use this tool to list and filter serverless spark sessions.
```

## Response Format

```json
{
  "sessions": [
    {
      "name": "projects/my-project/locations/us-central1/sessions/session-abc-123",
      "uuid": "a1b2c3d4-e5f6-7890-1234-567890abcdef",
      "state": "ACTIVE",
      "creator": "alice@example.com",
      "createTime": "2023-10-27T10:00:00Z",
      "consoleUrl": "https://console.cloud.google.com/dataproc/interactive/us-central1/session-abc-123/details?project=my-project",
      "logsUrl": "https://console.cloud.google.com/logs/viewer?..."
    },
    {
      "name": "projects/my-project/locations/us-central1/sessions/session-def-456",
      "uuid": "b2c3d4e5-f6a7-8901-2345-678901bcdefa",
      "state": "TERMINATED",
      "creator": "alice@example.com",
      "createTime": "2023-10-27T11:30:00Z",
      "consoleUrl": "https://console.cloud.google.com/dataproc/interactive/us-central1/session-def-456/details?project=my-project",
      "logsUrl": "https://console.cloud.google.com/logs/viewer?..."
    }
  ],
  "nextPageToken": "abcd1234"
}
```

## Reference

| **field**    | **type** | **required** | **description**                                    |
| ------------ | :------: | :----------: | -------------------------------------------------- |
| type         |  string  |     true     | Must be "serverless-spark-list-sessions".          |
| source       |  string  |     true     | Name of the source the tool should use.            |
| description  |  string  |     true     | Description of the tool that is passed to the LLM. |
| authRequired | string[] |    false     | List of auth services required to invoke this tool |
