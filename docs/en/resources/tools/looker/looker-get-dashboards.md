---
title: "looker-get-dashboards"
type: docs
weight: 1
description: >
  "looker-get-dashboards" tool searches for a saved Dashboard by name or description.
aliases:
- /resources/tools/looker-get-dashboards
---

## About

The `looker-get-dashboards` tool searches for a saved Dashboard by
name or description.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-get-dashboards` takes four parameters, the `title`, `desc`, `limit`
and `offset`.

Title and description use SQL style wildcards and are case insensitive.

Limit and offset are used to page through a larger set of matches and
default to 100 and 0.

## Example

```yaml
tools:
        get_dashboards:
            kind: looker-get-dashboards
            source: looker-source
            description: |
              get_dashboards Tool
    
              This tool searches for saved dashboards in a Looker instance. It returns a list of JSON objects, each representing a dashboard.
    
              Search Parameters:
              - title (optional): Filter by dashboard title (supports wildcards).
              - folder_id (optional): Filter by the ID of the folder where the dashboard is saved.
              - user_id (optional): Filter by the ID of the user who created the dashboard.
              - description (optional): Filter by description content (supports wildcards).
              - id (optional): Filter by specific dashboard ID.
              - limit (optional): Maximum number of results to return. Defaults to a system limit.
              - offset (optional): Starting point for pagination.
    
              String Search Behavior:
              - Case-insensitive matching.
              - Supports SQL LIKE pattern match wildcards:
                - `%`: Matches any sequence of zero or more characters. (e.g., `"finan%"` matches "financial", "finance")
                - `_`: Matches any single character. (e.g., `"s_les"` matches "sales")
              - Special expressions for null checks:
                - `"IS NULL"`: Matches dashboards where the field is null.
                - `"NOT NULL"`: Excludes dashboards where the field is null.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-get-dashboards"                    |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
