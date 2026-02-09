---
title: "cloud-logging-admin-list-log-names"
type: docs
description: >
  A "cloud-logging-admin-list-log-names" tool lists the log names in the project.
aliases:
- /resources/tools/cloud-logging-admin-list-log-names
---

## About

The `cloud-logging-admin-list-log-names` tool lists the log names available in the Google Cloud project.
It's compatible with the following sources:

- [cloud-logging-admin](../../sources/cloud-logging-admin.md)

## Example

```yaml
kind: tools
name: list_log_names
type: cloud-logging-admin-list-log-names
source: my-cloud-logging
description: Lists all log names in the project.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "cloud-logging-admin-list-log-names".      |
| source      |  string  |     true     | Name of the cloud-logging-admin source.            |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |

### Parameters

| **parameter** | **type** | **required** | **description** |
|:--------------|:--------:|:------------:|:----------------|
| limit | integer | false | Maximum number of log entries to return (default: 200). |
