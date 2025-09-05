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

          This tool creates a new dashboard in Looker. The dashboard is
          initially empty and the add_dashboard_element tool is used to
          add content to the dashboard.

          The newly created dashboard will be created in the user's
          personal folder in looker. The dashboard name must be unique.

          The result is a json document with a link to the newly
          created dashboard and the id of the dashboard. Use the id
          when calling add_dashboard_element.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| type        |                   string                   |     true     | Must be "looker-make-dashboard"                                                                  |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
