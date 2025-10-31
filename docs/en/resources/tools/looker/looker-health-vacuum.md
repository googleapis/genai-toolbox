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

| **field**   | **type** | **required** | **description**                                                                   |
|:------------|:---------|:-------------|:----------------------------------------------------------------------------------|
| action      | string   | true         | The vacuum to perform: `models`, or `explores`.                                   |
| project     | string   | false        | The name of the Looker project to vacuum.                                         |
| model       | string   | false        | The name of the Looker model to vacuum.                                           |
| explore     | string   | false        | The name of the Looker explore to vacuum.                                         |
| timeframe   | int      | false        | The timeframe in days to analyze for usage. Defaults to 90.                       |
| min_queries | int      | false        | The minimum number of queries for an object to be considered used. Defaults to 1. |

## Example

Identify unnused fields (*in this case, less than 1 query in the last 20 days*) and joins in the `order_items` explore and `thelook` model

```yaml
tools:
  health_vacuum:
    kind: looker-health-vacuum
    source: looker-source
    description: |
      health-vacuum Tool

      This tool suggests models or explores that can removed
      because they are unused.

      It accepts 6 parameters:
        1. `action`: can be "models" or "explores"
        2. `project`: the project to vacuum (optional)
        3. `model`: the model to vacuum (optional)
        4. `explore`: the explore to vacuum (optional)
        5. `timeframe`: the lookback period in days, default is 90
        6. `min_queries`: the minimum number of queries to consider a resource as active, default is 1

      The result is a list of objects that are candidates for deletion.
```


| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-health-vacuum"                     |
| source      |  string  |     true     | Looker source name                                 |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |