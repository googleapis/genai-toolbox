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

The `looker-health-vacuum` tool helps you identify unused LookML objects such as
models, explores, joins, and fields. The `action` parameter selects the type of
vacuum to perform:

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

Identify unnused fields (*in this case, less than 1 query in the last 20 days*)
and joins in the `order_items` explore and `thelook` model

```yaml
tools:
  health_vacuum:
    kind: looker-health-vacuum
    source: looker-source
    description: |
      health_vacuum Tool

      This tool identifies and suggests LookML models or explores that can be
      safely removed due to inactivity or low usage.

      Parameters:
      - action (required): The type of resource to analyze for removal candidates. Can be `"models"` or `"explores"`.
      - project (optional): The specific project ID to consider.
      - model (optional): The specific model name to consider. Requires `project` if used without `explore`.
      - explore (optional): The specific explore name to consider. Requires `model` if used.
      - timeframe (optional): The lookback period in days to assess usage. Defaults to `90` days.
      - min_queries (optional): The minimum number of queries for a resource to be considered active. Defaults to `1`.

      Output:
      A JSON array of objects, each representing a model or explore that is a candidate for deletion due to low usage.
```

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-health-vacuum"                     |
| source      |  string  |     true     | Looker source name                                 |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
