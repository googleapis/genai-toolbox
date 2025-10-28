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
          Query Tool

          This tool is used to run a query against the LookML model. The
          model, explore, and fields list must be specified. Pivots,
          filters and sorts are optional.

          The model can be found from the get_models tool. The explore
          can be found from the get_explores tool passing in the model.
          The fields can be found from the get_dimensions, get_measures,
          get_filters, and get_parameters tools, passing in the model
          and the explore.

          Provide a model_id and explore_name, then a list
          of fields. Optionally a list of pivots can be provided.
          The pivots must also be included in the fields list.

          Filters are provided as a map of {"field.id": "condition",
          "field.id2": "condition2", ...}. Do not put the field.id in
          quotes. Filter expressions can be found at
          https://cloud.google.com/looker/docs/filter-expressions.
          If the condition is a string that contains a comma, use a second
          set of quotes. For example, {"user.city": "'New York, NY'"}.

          Sorts can be specified like [ "field.id desc 0" ].

          An optional row limit can be added. If not provided the limit
          will default to 500. "-1" can be specified for unlimited.

          An optional query timezone can be added. The query_timezone to
          will default to that of the workstation where this MCP server
          is running, or Etc/UTC if that can't be determined. Not all
          models support custom timezones.

          The result of the query tool is JSON
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "looker-query"                                                                           |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
