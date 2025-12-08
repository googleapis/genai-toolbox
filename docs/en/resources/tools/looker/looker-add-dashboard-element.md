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

The `looker-add-dashboard-element` tool creates a new tile (element) within an existing Looker dashboard.
Tiles are added in the order this tool is called for a given `dashboard_id`.

CRITICAL ORDER OF OPERATIONS:
1. Create the dashboard using `make_dashboard`.
2. Add any dashboard-level filters using `add_dashboard_filter`.
3. Then, add elements (tiles) using this tool.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

## Example

```yaml
tools:
    add_dashboard_element:
        kind: looker-add-dashboard-element
        source: looker-source
        description: |
          add_dashboard_element Tool

          This tool creates a new tile (element) within an existing Looker dashboard.
          Tiles are added in the order this tool is called for a given `dashboard_id`.

          CRITICAL ORDER OF OPERATIONS:
          1. Create the dashboard using `make_dashboard`.
          2. Add any dashboard-level filters using `add_dashboard_filter`.
          3. Then, add elements (tiles) using this tool.

          Required Parameters:
          - dashboard_id: The ID of the target dashboard, obtained from `make_dashboard`.
          - model_name, explore_name, fields: These query parameters are inherited
            from the `query` tool and are required to define the data for the tile.

          Optional Parameters:
          - title: An optional title for the dashboard tile.
          - pivots, filters, sorts, limit, query_timezone: These query parameters are
            inherited from the `query` tool and can be used to customize the tile's query.
          - vis_config: A JSON object defining the visualization settings for this tile.
            The structure and options are the same as for the `query_url` tool's `vis_config`.

          Connecting to Dashboard Filters:
          A dashboard element can be connected to one or more dashboard filters (created with
          `add_dashboard_filter`). To do this, specify the `name` of the dashboard filter
          and the `field` from the element's query that the filter should apply to.
          The format for specifying the field is `view_name.field_name`.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|:------------|:--------:|:------------:|----------------------------------------------------|
| kind        | string   | true         | Must be "looker-add-dashboard-element".            |
| source      | string   | true         | Name of the source the SQL should execute on.      |
| description | string   | true         | Description of the tool that is passed to the LLM. |