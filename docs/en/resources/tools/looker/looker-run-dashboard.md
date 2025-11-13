---
title: "looker-run-dashboard"
type: docs
weight: 1
description: >
  "looker-run-dashboard" runs the queries associated with a dashboard.
aliases:
- /resources/tools/looker-run-dashboard
---

## About

The `looker-run-dashboard` tool runs the queries associated with a
dashboard.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-run-dashboard` takes one parameter, the `dashboard_id`.

## Example

```yaml
tools:
    run_dashboard:
        kind: looker-run-dashboard
        source: looker-source
        description: |
          run_dashboard Tool

          This tools runs the query associated with each tile in a dashboard
          and returns the data in a JSON structure. It accepts the dashboard_id
          as the parameter.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-run-dashboard"                     |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |