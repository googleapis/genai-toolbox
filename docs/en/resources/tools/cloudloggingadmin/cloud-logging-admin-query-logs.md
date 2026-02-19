---
title: "cloud-logging-admin-query-logs"
type: docs
description: >
  A "cloud-logging-admin-query-logs" tool queries log entries.
aliases:
- /resources/tools/cloud-logging-admin-query-logs
---

## About

The `cloud-logging-admin-query-logs` tool allows you to query log entries from Google Cloud Logging using the advanced logs filter syntax.
It's compatible with the following sources:

- [cloud-logging-admin](../../sources/cloud-logging-admin.md)

## Example

```yaml
kind: tools
name: query_logs
type: cloud-logging-admin-query-logs
source: my-cloud-logging
description: Queries log entries from Cloud Logging.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "cloud-logging-admin-query-logs".          |
| source      |  string  |     true     | Name of the cloud-logging-admin source.            |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |

### Parameters

| **parameter** | **type** | **required** | **description** |
|:--------------|:--------:|:------------:|:----------------|
| filter | string | false | Cloud Logging filter query. Common fields: resource.type, resource.labels.*, logName, severity, textPayload, jsonPayload.*, protoPayload.*, labels.*, httpRequest.*. Operators: =, !=, <, <=, >, >=, :, =~, AND, OR, NOT. |
| newestFirst | boolean | false | Set to true for newest logs first. Defaults to oldest first. |
| startTime | string | false | Start time in RFC3339 format (e.g., 2025-12-09T00:00:00Z). Defaults to 30 days ago. |
| endTime | string | false | End time in RFC3339 format (e.g., 2025-12-09T23:59:59Z). Defaults to now. |
| verbose | boolean | false | Include additional fields (insertId, trace, spanId, httpRequest, labels, operation, sourceLocation). Defaults to false. |
| limit | integer | false | Maximum number of log entries to return. Default: `200`. |
