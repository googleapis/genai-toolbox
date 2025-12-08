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

          This tool executes the queries associated with each tile in a specified dashboard
          and returns the aggregated data in a JSON structure.

          Parameters:
          - dashboard_id (required): The unique identifier of the dashboard to run,
            typically obtained from the `get_dashboards` tool.

          Output:
          The data from all dashboard tiles is returned as a JSON object.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-run-dashboard"                     |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
