---
title: "looker-add-dashboard-filter"
type: docs
weight: 1
description: >
  The "looker-add-dashboard-filter" tool adds a filter to a specified dashboard.
aliases:
- /resources/tools/looker-add-dashboard-filter
---

## About

The `looker-add-dashboard-filter` tool adds a filter to a specified Looker dashboard.

CRITICAL ORDER OF OPERATIONS:
1. Create a dashboard using `make_dashboard`.
2. Add all desired filters using this tool (`add_dashboard_filter`).
3. Finally, add dashboard elements (tiles) using `add_dashboard_element`.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

## Parameters

| **parameter** | **type** | **required** | **default** | **description**                                                                                                                                          |
|:--------------|:--------:|:------------:|:-----------:|----------------------------------------------------------------------------------------------------------------------------------------------------------|
| dashboard_id  |  string  |     true     |    none     | The ID of the dashboard to add the filter to, obtained from `make_dashboard`.                                                                            |
| name          |  string  |     true     |    none     | A unique internal identifier for the filter. This name is used later in `add_dashboard_element` to bind tiles to this filter.                            |
| title         |  string  |     true     |    none     | The label displayed to users in the Looker UI.                                                                                                           |
| flter_type    |  string  |     true     |    none     | The filter type of filter. Can be `date_filter`, `number_filter`, `string_filter`, or `field_filter`.                                                    |
| default_value |  string  |    false     |    none     | The initial value for the filter.                                                                                                                        |
| model         |  string  |    false     |    none     | Required if `filter_type` is `field_filter`. The name of the LookML model, obtained from `get_models`.                                                   |
| explore       |  string  |    false     |    none     | Required if `filter_type` is `field_filter`. The name of the explore within the model, obtained from `get_explores`.                                     |
| dimension     |  string  |    false     |    none     | Required if `filter_type` is `field_filter`. The name of the field (e.g., `view_name.field_name`) to base the filter on, obtained from `get_dimensions`. |

## Example

```yaml
tools:
  add_dashboard_filter:
    kind: looker-add-dashboard-filter
    source: looker-source
    description: |
      This tool adds a filter to a Looker dashboard.

      CRITICAL ORDER OF OPERATIONS:
      1. Create a dashboard using `make_dashboard`.
      2. Add all desired filters using this tool (`add_dashboard_filter`).
      3. Finally, add dashboard elements (tiles) using `add_dashboard_element`.

      Parameters:
      - dashboard_id (required): The ID from `make_dashboard`.
      - name (required): A unique internal identifier for the filter. You will use this `name` later in `add_dashboard_element` to bind tiles to this filter.
      - title (required): The label displayed to users in the UI.
      - filter_type (required): One of `date_filter`, `number_filter`, `string_filter`, or `field_filter`.
      - default_value (optional): The initial value for the filter.

      Field Filters (`flter_type: field_filter`):
      If creating a field filter, you must also provide:
      - model
      - explore
      - dimension
      The filter will inherit suggestions and type information from this LookML field.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-add-dashboard-filter".             |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |