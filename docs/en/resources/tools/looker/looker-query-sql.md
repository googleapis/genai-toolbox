---
title: "looker-query-sql"
type: docs
weight: 1
description: >
  "looker-query-sql" generates a sql query using the Looker
  semantic model.
aliases:
- /resources/tools/looker-query-sql
---

## About

The `looker-query-sql` generates a sql query using the Looker
semantic model.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-query-sql` takes eight parameters:

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
    query_sql:
        kind: looker-query-sql
        source: looker-source
        description: |
          This tool generates the underlying SQL query that Looker would execute
          against the database for a given set of parameters. It is useful for
          understanding how Looker translates a request into SQL.

          Parameters:
          All parameters for this tool are identical to those of the `query` tool.
          This includes `model_name`, `explore_name`, `fields` (required),
          and optional parameters like `pivots`, `filters`, `sorts`, `limit`, and `query_timezone`.

          Output:
          The result of this tool is the raw SQL text.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-query-sql"                         |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
