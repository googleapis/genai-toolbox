---
title: "looker-health-pulse"
type: docs
weight: 1
description: >
  "looker-health-pulse" performs health checks on a Looker instance, with multiple actions available (e.g., checking database connections, dashboard performance, etc).
aliases:
- /resources/tools/looker-health-pulse
---

## About

The `looker-health-pulse` tool performs health checks on a Looker instance. The `action` parameter selects the type of check to perform:

- `check_db_connections`: Checks all database connections, runs supported tests, and reports query counts.
- `check_dashboard_performance`: Finds dashboards with slow running queries in the last 7 days.
- `check_dashboard_errors`: Lists dashboards with erroring queries in the last 7 days.
- `check_explore_performance`: Lists the slowest explores in the last 7 days and reports average query runtime.
- `check_schedule_failures`: Lists schedules that have failed in the last 7 days.
- `check_legacy_features`: Lists enabled legacy features. (*To note, this function is not
  available in Looker Core.*)

## Parameters

| **field**     | **type** | **required** | **description**                             |
|---------------|:--------:|:------------:|---------------------------------------------|
| action        | string   | true         | The health check to perform                 |


| **action**                | **description**                                                                |
|---------------------------|--------------------------------------------------------------------------------|
| check_db_connections      | Checks all database connections and reports query counts and errors            |
| check_dashboard_performance | Finds dashboards with slow queries (>30s) in the last 7 days                 |
| check_dashboard_errors    | Lists dashboards with erroring queries in the last 7 days                      |
| check_explore_performance | Lists slowest explores and average query runtime                               |
| check_schedule_failures   | Lists failed schedules in the last 7 days                                      |
| check_legacy_features     | Lists enabled legacy features                                                  |

## Example

```yaml
tools:
  health_pulse:
    kind: looker-health-pulse
    source: looker-source
    description: |
      health-pulse Tool

      This tool takes the pulse of a Looker instance by taking
      one of the following actions:
        1. `check_db_connections`,
        2. `check_dashboard_performance`,
        3. `check_dashboard_errors`,
        4. `check_explore_performance`,
        5. `check_schedule_failures`, or
        6. `check_legacy_features`
      
      The `check_legacy_features` action is only available in Looker Core. If
      it is called on a Looker Core instance, you will get a notice. That notice
      should not be reported as an error.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-health-pulse"                      |
| source      |  string  |     true     | Looker source name                                 |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |