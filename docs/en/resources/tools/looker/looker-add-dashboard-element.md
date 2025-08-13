---
title: "looker-add-dashboard-element"
type: docs
weight: 1
description: >
  "looker-add-dashboard-element" creates a dashboard element in the given dashboard.
aliases:
- /resources/tools/looker-add-dashboard-element
---

## About

The `looker-add-dashboard-element` creates a dashboard element
in the given dashboard.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-add-dashboard-element` takes eleven parameters:

1. the `model`
2. the `explore`
3. the `fields` list
4. an optional set of `filters`
5. an optional set of `pivots`
6. an optional set of `sorts`
7. an optional `limit`
8. an optional `tz`
9. an optional `vis_config`
10. the `title`
11. the `dashboard_id`

## Example

```yaml
tools:
    add_dashboard_element:
        kind: looker-add-dashboard-element
        source: looker-source
        description: |
          add_dashboard_element Tool

          This tool creates a new tile in a Looker dashboard using
          the query parameters and the vis_config specified.

          Most of the parameters are the same as the query_url
          tool. In addition, there is a title that may be provided.
          The dashboard_id must be specified. That is obtained
          from calling make_dashboard.

          This tool can be called many times for one dashboard_id
          and the resulting tiles will be added in order.

          A dashboard element can be connected to one or more
          dashboard filters by specifying the filter name and the
          field to which the filter should apply formatted as
          `view_name.field_name`.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-add-dashboard-element"             |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
