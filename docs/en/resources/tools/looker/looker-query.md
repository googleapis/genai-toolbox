---
title: "looker-query"
type: docs
weight: 1
description: >
  "looker-query" runs an inline query using the Looker
  semantic model.
aliases:
- /resources/tools/looker-query
---

## About

The `looker-query` tool runs a query using the Looker
semantic model.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-query` takes eight parameters:

1. the `model`
2. the `explore`
3. the `fields` list
4. an optional set of `filters`
5. an optional set of `pivots`
6. an optional set of `sorts`
7. an optional `limit`
8. an optional `tz`

Starting in Looker v25.18, these queries can be identified in Looker's
System Activity. In the History explore, use the field API Client Name
to find MCP Toolbox queries.

## Example

```yaml
tools:
    query:
        kind: looker-query
        source: looker-source
        description: |
          This tool runs a query against a LookML model and returns the results in JSON format.

          Required Parameters:
          - model_name: The name of the LookML model (from `get_models`).
          - explore_name: The name of the explore (from `get_explores`).
          - fields: A list of field names (dimensions, measures, filters, or parameters) to include in the query.

          Optional Parameters:
          - pivots: A list of fields to pivot the results by. These fields must also be included in the `fields` list.
          - filters: A map of filter expressions, e.g., `{"view.field": "value", "view.date": "7 days"}`.
            - Do not quote field names.
            - Use `not null` instead of `-NULL`.
            - If a value contains a comma, enclose it in single quotes (e.g., "'New York, NY'").
          - sorts: A list of fields to sort by, optionally including direction (e.g., `["view.field desc"]`).
          - limit: Row limit (default 500). Use "-1" for unlimited.
          - query_timezone: specific timezone for the query (e.g. `America/Los_Angeles`).

          Note: Use `get_dimensions`, `get_measures`, `get_filters`, and `get_parameters` to find valid fields.

          The result of the query tool is JSON
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-query"                             |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
