---
title: "looker-get-models"
type: docs
weight: 1
description: > 
  A "looker-get-models" tool returns all the models in the source.
aliases:
- /resources/tools/looker-get-models
---

## About

A `looker-get-models` tool returns all the models the source.

It's compatible with the following sources:

- [looker](../../sources/looker/)

`looker-get-models` accepts no parameters.

## Example

```yaml
tools:
    get_models:
        kind: looker-get-models
        source: looker-source
        description: |
          The get_models tool retrieves the list of LookML models in the Looker system.

          It takes no parameters.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "looker-get-models".                                                                     |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
