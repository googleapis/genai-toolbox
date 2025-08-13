---
title: "looker-add-dashboard-filter"
type: docs
weight: 1
description: >
  "looker-add-dashboard-filter" creates a dashboard filter in the given dashboard.
aliases:
- /resources/tools/looker-add-dashboard-filter
---

## About

The `looker-add-dashboard-filter` creates a dashboard filter
in the given dashboard.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-add-dashboard-element` takes eleven parameters:

1. the `dashboard_id`
2. the `name`
3. the `title`
4. the `type`. date, string, number, or field
5. an optional `default_value`
6. an optional `model`
7. an optional `explore`
8. an optional `dimension`
9. the `allow_multiple_values` flag
10. the `required` flag



## Example

```yaml
tools:
    add_dashboard_filter:
        kind: looker-add-dashboard-filter
        source: looker-source
        description: |
          add_dashboard_filter Tool

          This tool will add a filter to a dashboard. The name of the
          filter will be used to refer to the filter when creating
          dashboard elements. The title will be displayed with the
          filter in the Looker UI.

          The type of the filter is the datatype. If it is `field_filter` then
          the type will be inherited from a field in the model, and that
          field will be used to suggest filter values. Otherwise the value
          of type can be `date_filter`, `number_filter`, or `string_filter`.

          The default value is optional.

          The parameters model, explore, and dimension should only be
          specified if the type is `field_filter`.

          Add any dashboard filters after creating the dashboard but
          before adding any dashboard elements. When adding dashboard
          elements specify the name of the dashboad filter and the
          field of that element's query to which the filter should
          apply.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-add-dashboard-filter"              |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
