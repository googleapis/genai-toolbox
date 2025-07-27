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

- [looker](../../sources/looker/)

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
          get_looks Tool

          This tool is used to search for saved looks in a Looker instance.
          String search params use case-insensitive matching. String search
          params can contain % and '_' as SQL LIKE pattern match wildcard
          expressions. example="dan%" will match "danger" and "Danzig" but
          not "David" example="D_m%" will match "Damage" and "dump".

          Most search params can accept "IS NULL" and "NOT NULL" as special
          expressions to match or exclude (respectively) rows where the
          column is null.

          The limit and offset are used to paginate the results.

          The result of the get_looks tool is a list of json objects.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "looker-get-looks"                                                                       |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
