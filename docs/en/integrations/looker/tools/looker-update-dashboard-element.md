---
title: "looker-update-dashboard-element"
type: docs
weight: 1
description: >
  "looker-update-dashboard-element" updates an existing element in a Looker dashboard.
---

## About

The `looker-update-dashboard-element` tool updates an existing query element (tile) in a Looker dashboard. It reconstructs the query for the element using the provided parameters and updates its title and visualization configuration.

## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: update_dashboard_element
type: looker-update-dashboard-element
source: looker-source
description: |
  This tool updates an existing query element (tile) in a Looker dashboard.
  It reconstructs the query for the element using the provided parameters
  and updates its title and visualization configuration.

  Required Parameters:
  - dashboard_id: The ID of the dashboard containing the element.
  - dashboard_element_id: The ID of the element to update.
  - model, explore, fields: These query parameters define the data for the tile.

  Optional Parameters:
  - title: The new title for the dashboard tile.
  - pivots, filters, sorts, limit, tz: These query parameters customize the tile's query.
  - vis_config: A JSON object defining the visualization settings.
  - dashboard_filters: An array of dashboard filters to connect to this element.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|:------------|:--------:|:------------:|----------------------------------------------------|
| type        | string   | true         | Must be "looker-update-dashboard-element".        |
| source      | string   | true         | Name of the Looker source.                         |
| description | string   | true         | Description of the tool that is passed to the LLM. |
