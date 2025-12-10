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

The `looker-health-pulse` tool performs health checks on a Looker instance. The
`action` parameter selects the type of check to perform:

- `check_db_connections`: Checks all database connections, runs supported tests,
  and reports query counts.
- `check_dashboard_performance`: Finds dashboards with slow running queries in
  the last 7 days.
- `check_dashboard_errors`: Lists dashboards with erroring queries in the last 7
  days.
- `check_explore_performance`: Lists the slowest explores in the last 7 days and
  reports average query runtime.
- `check_schedule_failures`: Lists schedules that have failed in the last 7
  days.
- `check_legacy_features`: Lists enabled legacy features. (*To note, this
  function is not available in Looker Core.*)

## Parameters

| **field** | **type** | **required** | **description**             |
|-----------|:--------:|:------------:|-----------------------------|
| action    |  string  |     true     | The health check to perform |

| **action**                  | **description**                                                     |
|-----------------------------|---------------------------------------------------------------------|
| check_db_connections        | Checks all database connections and reports query counts and errors |
| check_dashboard_performance | Finds dashboards with slow queries (>30s) in the last 7 days        |
| check_dashboard_errors      | Lists dashboards with erroring queries in the last 7 days           |
| check_explore_performance   | Lists slowest explores and average query runtime                    |
| check_schedule_failures     | Lists failed schedules in the last 7 days                           |
| check_legacy_features       | Lists enabled legacy features                                       |

## Example

```yaml
tools:
  health_pulse:
    kind: looker-health-pulse
    source: looker-source
    description: |
      This tool performs various health checks on a Looker instance.

      Parameters:
      - action (required): Specifies the type of health check to perform.
        Choose one of the following:
        - `check_db_connections`: Verifies database connectivity.
        - `check_dashboard_performance`: Assesses dashboard loading performance.
        - `check_dashboard_errors`: Identifies errors within dashboards.
        - `check_explore_performance`: Evaluates explore query performance.
        - `check_schedule_failures`: Reports on failed scheduled deliveries.
        - `check_legacy_features`: Checks for the usage of legacy features.

      Note on `check_legacy_features`:
      This action is exclusively available in Looker Core instances. If invoked
      on a non-Looker Core instance, it will return a notice rather than an error.
      This notice should be considered normal behavior and not an indication of an issue.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-health-pulse"                      |
| source      |  string  |     true     | Looker source name                                 |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
