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
9. an optional `vis_config`

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
          are the same as the query tool with an additional vis_config
          parameter.

          The vis_config is optional. If provided, it will be used to
          control the default visualization for the query. These are
          some sample vis_config settings.

          A bar chart -
          {{
            "defaults_version": 1,
            "label_density": 25,
            "legend_position": "center",
            "limit_displayed_rows": false,
            "ordering": "none",
            "plot_size_by_field": false,
            "point_style": "none",
            "show_null_labels": false,
            "show_silhouette": false,
            "show_totals_labels": false,
            "show_value_labels": false,
            "show_view_names": false,
            "show_x_axis_label": true,
            "show_x_axis_ticks": true,
            "show_y_axis_labels": true,
            "show_y_axis_ticks": true,
            "stacking": "normal",
            "totals_color": "#808080",
            "trellis": "",
            "type": "looker_bar",
            "x_axis_gridlines": false,
            "x_axis_reversed": false,
            "x_axis_scale": "auto",
            "x_axis_zoom": true,
            "y_axis_combined": true,
            "y_axis_gridlines": true,
            "y_axis_reversed": false,
            "y_axis_scale_mode": "linear",
            "y_axis_tick_density": "default",
            "y_axis_tick_density_custom": 5,
            "y_axis_zoom": true
          }}

          A column chart with an option advanced_vis_config -
          {{
            "advanced_vis_config": "{ chart: { type: 'pie', spacingBottom: 50, spacingLeft: 50, spacingRight: 50, spacingTop: 50, }, legend: { enabled: false, }, plotOptions: { pie: { dataLabels: { enabled: true, format: '\u003cb\u003e{key}\u003c/b\u003e\u003cspan style=\"font-weight: normal\"\u003e - {percentage:.2f}%\u003c/span\u003e', }, showInLegend: false, }, }, series: [], }",
            "colors": [
              "grey"
            ],
            "defaults_version": 1,
            "hidden_fields": [],
            "label_density": 25,
            "legend_position": "center",
            "limit_displayed_rows": false,
            "note_display": "below",
            "note_state": "collapsed",
            "note_text": "Unsold inventory only",
            "ordering": "none",
            "plot_size_by_field": false,
            "point_style": "none",
            "series_colors": {},
            "show_null_labels": false,
            "show_silhouette": false,
            "show_totals_labels": false,
            "show_value_labels": true,
            "show_view_names": false,
            "show_x_axis_label": true,
            "show_x_axis_ticks": true,
            "show_y_axis_labels": true,
            "show_y_axis_ticks": true,
            "stacking": "normal",
            "totals_color": "#808080",
            "trellis": "",
            "type": "looker_column",
            "x_axis_gridlines": false,
            "x_axis_reversed": false,
            "x_axis_scale": "auto",
            "x_axis_zoom": true,
            "y_axes": [],
            "y_axis_combined": true,
            "y_axis_gridlines": true,
            "y_axis_reversed": false,
            "y_axis_scale_mode": "linear",
            "y_axis_tick_density": "default",
            "y_axis_tick_density_custom": 5,
            "y_axis_zoom": true
          }}

          A line chart -
          {{
            "defaults_version": 1,
            "hidden_pivots": {},
            "hidden_series": [],
            "interpolation": "linear",
            "label_density": 25,
            "legend_position": "center",
            "limit_displayed_rows": false,
            "plot_size_by_field": false,
            "point_style": "none",
            "series_types": {},
            "show_null_points": true,
            "show_value_labels": false,
            "show_view_names": false,
            "show_x_axis_label": true,
            "show_x_axis_ticks": true,
            "show_y_axis_labels": true,
            "show_y_axis_ticks": true,
            "stacking": "",
            "trellis": "",
            "type": "looker_line",
            "x_axis_gridlines": false,
            "x_axis_reversed": false,
            "x_axis_scale": "auto",
            "y_axis_combined": true,
            "y_axis_gridlines": true,
            "y_axis_reversed": false,
            "y_axis_scale_mode": "linear",
            "y_axis_tick_density": "default",
            "y_axis_tick_density_custom": 5
          }}

          An area chart -
          {{
            "defaults_version": 1,
            "interpolation": "linear",
            "label_density": 25,
            "legend_position": "center",
            "limit_displayed_rows": false,
            "plot_size_by_field": false,
            "point_style": "none",
            "series_types": {},
            "show_null_points": true,
            "show_silhouette": false,
            "show_totals_labels": false,
            "show_value_labels": false,
            "show_view_names": false,
            "show_x_axis_label": true,
            "show_x_axis_ticks": true,
            "show_y_axis_labels": true,
            "show_y_axis_ticks": true,
            "stacking": "normal",
            "totals_color": "#808080",
            "trellis": "",
            "type": "looker_area",
            "x_axis_gridlines": false,
            "x_axis_reversed": false,
            "x_axis_scale": "auto",
            "x_axis_zoom": true,
            "y_axis_combined": true,
            "y_axis_gridlines": true,
            "y_axis_reversed": false,
            "y_axis_scale_mode": "linear",
            "y_axis_tick_density": "default",
            "y_axis_tick_density_custom": 5,
            "y_axis_zoom": true
          }}

          A scatter plot -
          {{
            "cluster_points": false,
            "custom_quadrant_point_x": 5,
            "custom_quadrant_point_y": 5,
            "custom_value_label_column": "",
            "custom_x_column": "",
            "custom_y_column": "",
            "defaults_version": 1,
            "hidden_fields": [],
            "hidden_pivots": {},
            "hidden_points_if_no": [],
            "hidden_series": [],
            "interpolation": "linear",
            "label_density": 25,
            "legend_position": "center",
            "limit_displayed_rows": false,
            "limit_displayed_rows_values": {
              "first_last": "first",
              "num_rows": 0,
              "show_hide": "hide"
            },
            "plot_size_by_field": false,
            "point_style": "circle",
            "quadrant_properties": {
              "0": {
                "color": "",
                "label": "Quadrant 1"
              },
              "1": {
                "color": "",
                "label": "Quadrant 2"
              },
              "2": {
                "color": "",
                "label": "Quadrant 3"
              },
              "3": {
                "color": "",
                "label": "Quadrant 4"
              }
            },
            "quadrants_enabled": false,
            "series_labels": {},
            "series_types": {},
            "show_null_points": false,
            "show_value_labels": false,
            "show_view_names": true,
            "show_x_axis_label": true,
            "show_x_axis_ticks": true,
            "show_y_axis_labels": true,
            "show_y_axis_ticks": true,
            "size_by_field": "roi",
            "stacking": "normal",
            "swap_axes": true,
            "trellis": "",
            "type": "looker_scatter",
            "x_axis_gridlines": false,
            "x_axis_reversed": false,
            "x_axis_scale": "auto",
            "x_axis_zoom": true,
            "y_axes": [
              {
                "label": "",
                "orientation": "bottom",
                "series": [
                  {
                    "axisId": "Channel_0 - average_of_roi_first",
                    "id": "Channel_0 - average_of_roi_first",
                    "name": "Channel_0"
                  },
                  {
                    "axisId": "Channel_1 - average_of_roi_first",
                    "id": "Channel_1 - average_of_roi_first",
                    "name": "Channel_1"
                  },
                  {
                    "axisId": "Channel_2 - average_of_roi_first",
                    "id": "Channel_2 - average_of_roi_first",
                    "name": "Channel_2"
                  },
                  {
                    "axisId": "Channel_3 - average_of_roi_first",
                    "id": "Channel_3 - average_of_roi_first",
                    "name": "Channel_3"
                  },
                  {
                    "axisId": "Channel_4 - average_of_roi_first",
                    "id": "Channel_4 - average_of_roi_first",
                    "name": "Channel_4"
                  }
                ],
                "showLabels": true,
                "showValues": true,
                "tickDensity": "custom",
                "tickDensityCustom": 100,
                "type": "linear",
                "unpinAxis": false
              }
            ],
            "y_axis_combined": true,
            "y_axis_gridlines": true,
            "y_axis_reversed": false,
            "y_axis_scale_mode": "linear",
            "y_axis_tick_density": "default",
            "y_axis_tick_density_custom": 5,
            "y_axis_zoom": true
          }}

          A single value visualization -
          {{
            "defaults_version": 1,
            "show_view_names": false,
            "type": "looker_single_record"
          }}

          A Pie chart -
          {{
            "defaults_version": 1,
            "label_density": 25,
            "label_type": "labPer",
            "legend_position": "center",
            "limit_displayed_rows": false,
            "ordering": "none",
            "plot_size_by_field": false,
            "point_style": "none",
            "series_types": {},
            "show_null_labels": false,
            "show_silhouette": false,
            "show_totals_labels": false,
            "show_value_labels": false,
            "show_view_names": false,
            "show_x_axis_label": true,
            "show_x_axis_ticks": true,
            "show_y_axis_labels": true,
            "show_y_axis_ticks": true,
            "stacking": "",
            "totals_color": "#808080",
            "trellis": "",
            "type": "looker_pie",
            "value_labels": "legend",
            "x_axis_gridlines": false,
            "x_axis_reversed": false,
            "x_axis_scale": "auto",
            "y_axis_combined": true,
            "y_axis_gridlines": true,
            "y_axis_reversed": false,
            "y_axis_scale_mode": "linear",
            "y_axis_tick_density": "default",
            "y_axis_tick_density_custom": 5
          }}

          The result is a JSON object with the id, slug, the url, and
          the long_url.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "looker-query-url"                                                                       |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
