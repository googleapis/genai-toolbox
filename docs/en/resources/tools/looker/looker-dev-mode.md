---
title: "looker-dev-mode"
type: docs
weight: 1
description: >
  A "looker-dev-mode" tool changes the current session into and out of dev mode
aliases:
- /resources/tools/looker-dev-mode
---

## About

A `looker-dev-mode` tool changes the session into and out of dev mode.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-dev-mode` accepts a boolean parameter, true to enter dev mode and false to exit dev mode.


## Example

```yaml
tools:
    dev_mode:
        kind: looker-dev-mode
        source: looker-source
        description: |
          dev_mode Tool

          Passing true to this tool switches the session to dev mode. Passing false to this tool switches the
          session to production mode.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "looker-dev-mode".                                                                       |
| source      |                   string                   |     true     | Name of the source Looker instance.                                                              |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |