---
title: "looker-health-vacuum"
type: docs
weight: 1
description: >
  "looker-health-vacuum" provides a set of commands to audit and identify unused LookML objects in a Looker instance.
aliases:
- /resources/tools/looker-health-vacuum
---

## About

The `looker-health-vacuum` tool helps you identify unused LookML objects such as models, explores, joins, and fields. The `action` parameter selects the type of vacuum to perform:

- `models`: Identifies unused explores within a model.
- `explores`: Identifies unused joins and fields within an explore.

## Parameters

| **field** | **type** | **required** | **description** |
| :--- | :--- | :--- | :--- |
| kind | string | true | Must be "looker-health-vacuum" |
| source | string | true | Looker source name |
| action | string | true | The vacuum to perform: `models`, or `explores`. |
| project | string | false | The name of the Looker project to vacuum. |
| model | string | false | The name of the Looker model to vacuum. |
| explore | string | false | The name of the Looker explore to vacuum. |
| timeframe | int | false | The timeframe in days to analyze for usage. Defaults to 90. |
| min_queries | int | false | The minimum number of queries for an object to be considered used. Defaults to 1. |

## Example

Identify unnused fields (*in this case, less than 1 query in the last 20 days*) and joins in the `order_items` explore and `thelook` model

```yaml
tools:
  vacuum-tool:
    kind: looker-health-vacuum
    source: looker-source
    description: |
      Vacuums the Looker instance by identifying unused explores, fields, and joins.
    parameters:
      action: explores
      project: "thelook_core"
      model: "thelook"
      explore: "order_items"
      timeframe: 20
      min_queries: 1
```

Identify unnused explores across all models in `thelook_core` project. 

```yaml
tools:
  vacuum-tool:
    kind: looker-health-vacuum
    source: looker-source
    description: |
      Vacuums the Looker instance by identifying unused explores, fields, and joins.
    parameters:
      action: models
      project: "thelook_core"
     