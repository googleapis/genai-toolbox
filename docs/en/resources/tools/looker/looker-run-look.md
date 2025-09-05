---
title: "looker-run-look"
type: docs
weight: 1
description: >
  "looker-run-look" runs the query associated with a saved Look.
aliases:
- /resources/tools/looker-run-look
---

## About

The `looker-run-look` tool runs the query associated with a
saved Look.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-run-look` takes one parameter, the `look_id`.

## Example

```yaml
tools:
    run_look:
        kind: looker-run-look
        source: looker-source
        description: |
          run_look Tool

          This tool runs the query associated with a look and returns
          the data in a JSON structure. It accepts the look_id as the
          parameter.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| type        |                   string                   |     true     | Must be "looker-run-look"                                                                        |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
