---
title: "cloud-logging-admin-list-resource-types"
type: docs
description: >
  A "cloud-logging-admin-list-resource-types" tool lists the monitored resource types.
aliases:
- /resources/tools/cloud-logging-admin-list-resource-types
---

## About

The `cloud-logging-admin-list-resource-types` tool lists the monitored resource types available in Google Cloud Logging.
It's compatible with the following sources:

- [cloud-logging-admin](../../sources/cloud-logging-admin.md)

## Example

```yaml
kind: tools
name: list_resource_types
type: cloud-logging-admin-list-resource-types
source: my-cloud-logging
description: Lists monitored resource types.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "cloud-logging-admin-list-resource-types".|
| source      |  string  |     true     | Name of the cloud-logging-admin source.            |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |

