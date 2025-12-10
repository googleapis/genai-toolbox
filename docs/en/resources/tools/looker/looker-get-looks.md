---
title: "looker-get-looks"
type: docs
weight: 1
description: >
  "looker-get-looks" searches for saved Looks in a Looker
  source.
aliases:
- /resources/tools/looker-get-looks
---

## About

The `looker-get-looks` tool searches for a saved Look by
name or description.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-get-looks` takes four parameters, the `title`, `desc`, `limit`
and `offset`.

Title and description use SQL style wildcards and are case insensitive.

Limit and offset are used to page through a larger set of matches and
default to 100 and 0.

## Example

```yaml
tools:
    get_looks:
        kind: looker-get-looks
        source: looker-source
        description: |
          This tool searches for saved Looks (pre-defined queries and visualizations)
          in a Looker instance. It returns a list of JSON objects, each representing a Look.

          Search Parameters:
          - title (optional): Filter by Look title (supports wildcards).
          - folder_id (optional): Filter by the ID of the folder where the Look is saved.
          - user_id (optional): Filter by the ID of the user who created the Look.
          - description (optional): Filter by description content (supports wildcards).
          - id (optional): Filter by specific Look ID.
          - limit (optional): Maximum number of results to return. Defaults to a system limit.
          - offset (optional): Starting point for pagination.

          String Search Behavior:
          - Case-insensitive matching.
          - Supports SQL LIKE pattern match wildcards:
            - `%`: Matches any sequence of zero or more characters. (e.g., `"dan%"` matches "danger", "Danzig")
            - `_`: Matches any single character. (e.g., `"D_m%"` matches "Damage", "dump")
          - Special expressions for null checks:
            - `"IS NULL"`: Matches Looks where the field is null.
            - `"NOT NULL"`: Excludes Looks where the field is null.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-get-looks"                         |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
