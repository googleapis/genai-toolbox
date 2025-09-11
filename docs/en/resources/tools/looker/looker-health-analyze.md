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

| **field** | **type** | **required** | **description** |
| :--- | :--- | :--- | :--- |
| kind | string | true | Must be "looker-health-analyze" |
| source | string | true | Looker source name |
| action | string | true | The analysis to perform: `projects`, `models`, or `explores`. |
| project | string | false | The name of the Looker project to analyze. |
| model | string | false | The name of the Looker model to analyze. Required for `explores` actions. |
| explore | string | false | The name of the Looker explore to analyze. Required for the `explores` action. |

## Example

```yaml
tools:
  analyze-tool:
    kind: looker-health-analyze
    source: looker-source
    description: |
      Analyzes Looker projects, models, and explores.
      Specify the `action` parameter to select the type of analysis.
    parameters:
      action: models
      project: "thelook"