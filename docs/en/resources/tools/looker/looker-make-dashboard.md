---
title: "looker-make-dashboard"
type: docs
weight: 1
description: >
  "looker-make-dashboard" generates a Looker dashboard in the users personal folder in
  Looker
aliases:
- /resources/tools/looker-make-dashboard
---

## About

The `looker-make-dashboard` creates a dashboard in the user's
Looker personal folder.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-make-dashboard` takes one parameter:

1. the `title`

## Example

```yaml
tools:
    make_dashboard:
        kind: looker-make-dashboard
        source: looker-source
        description: |
          make_dashboard Tool

          This tool creates a new, empty dashboard in Looker. Dashboards are stored
          in the user's personal folder, and the dashboard name must be unique.
          After creation, use `add_dashboard_filter` to add filters and
          `add_dashboard_element` to add content tiles.

          Required Parameters:
          - title (required): A unique title for the new dashboard.
          - description (required): A brief description of the dashboard's purpose.

          Output:
          A JSON object containing a link (`url`) to the newly created dashboard and
          its unique `id`. This `dashboard_id` is crucial for subsequent calls to
          `add_dashboard_filter` and `add_dashboard_element`.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-make-dashboard"                    |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
