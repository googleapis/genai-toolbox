---
title: "looker-query-url"
type: docs
weight: 1
description: >
  "looker-query-url" generates a url link to a Looker explore.
aliases:
- /resources/tools/looker-query-url
---

## About

The `looker-query-url` generates a url link to an explore in
Looker so the query can be investigated further.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-query-url` takes eight parameters:

1. the `model`
2. the `explore`
3. the `fields` list
4. an optional set of `filters`
5. an optional set of `pivots`
6. an optional set of `sorts`
7. an optional `limit`
8. an optional `tz`

## Example

```yaml
tools:
    query_url:
        kind: looker-query-url
        source: looker-source
        description: |
          Query URL Tool

          This tool is used to generate the URL of a query in Looker.
          The user can then explore the query further inside Looker.
          The tool also returns the query_id and slug. The parameters
          are the same as the `looker-query` tool.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "looker-query-url"                                                                       |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
