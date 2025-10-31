---
title: "looker-health-analyze"
type: docs
weight: 1
description: >
  "looker-health-analyze" provides a set of analytical commands for a Looker instance, allowing users to analyze projects, models, and explores.
aliases:
- /resources/tools/looker-health-analyze
---

## About

The `looker-health-analyze` tool performs various analysis tasks on a Looker instance. The `action` parameter selects the type of analysis to perform:

- `projects`: Analyzes all projects or a specified project, reporting on the number of models and view files, as well as Git connection and validation status.
- `models`: Analyzes all models or a specified model, providing a count of explores, unused explores, and total query counts.
- `explores`: Analyzes all explores or a specified explore, reporting on the number of joins, unused joins, fields, unused fields, and query counts. Being classified as **Unused** is determined by whether a field has been used as a field or filter within the past 90 days in production.

## Parameters

| **field**   | **type** | **required** | **description**                                                                            |
|:------------|:---------|:-------------|:-------------------------------------------------------------------------------------------|
| action      | string   | true         | The analysis to perform: `projects`, `models`, or `explores`.                              |
| project     | string   | false        | The name of the Looker project to analyze.                                                 |
| model       | string   | false        | The name of the Looker model to analyze. Required for `explores` actions.                  |
| explore     | string   | false        | The name of the Looker explore to analyze. Required for the `explores` action.             |
| timeframe   | int      | false        | The timeframe in days to analyze. Defaults to 90.                                          |
| min_queries | int      | false        | The minimum number of queries for a model or explore to be considered used. Defaults to 1. |

## Example

```yaml
tools:
  health_analyze:
    kind: looker-health-analyze
    source: looker-source
    description: |
      health-analyze Tool

      This tool calculates the usage of projects, models and explores.

      It accepts 6 parameters:
        1. `action`: can be "projects", "models", or "explores"
        2. `project`: the project to analyze (optional)
        3. `model`: the model to analyze (optional)
        4. `explore`: the explore to analyze (optional)
        5. `timeframe`: the lookback period in days, default is 90
        6. `min_queries`: the minimum number of queries to consider a resource as active, default is 1
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-health-analyze"                    |
| source      |  string  |     true     | Looker source name                                 |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |