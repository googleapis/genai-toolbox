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

`looker-dev-mode` accepts a boolean parameter, true to enter dev mode and false
to exit dev mode.

## Example

```yaml
tools:
    dev_mode:
        kind: looker-dev-mode
        source: looker-source
        description: |
          dev_mode Tool

          This tool allows toggling the Looker IDE session between Development Mode and Production Mode.
          Development Mode enables making and testing changes to LookML projects.

          Parameters:
          - enable (required): A boolean value.
            - `true`: Switches the current session to Development Mode.
            - `false`: Switches the current session to Production Mode.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-dev-mode".                         |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
