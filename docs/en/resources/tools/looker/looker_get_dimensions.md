---
title: "looker-get-dimensions"
type: docs
weight: 1
description: >
  A "looker-get-dimensions" tool returns all the dimensions from a given explore
  in a given model in the source.
aliases:
- /resources/tools/looker-get-dimensions
---

## About

A `looker-get-dimensions` tool returns all the dimensions from a given explore
in a given mode in the source.

It's compatible with the following sources:

- [looker](../../sources/looker/)

`looker-get-dimensions` accepts two parameters, the `model` and the `explore`.

## Example

```yaml
tools:
    get_dimensions:
        kind: looker-get-dimensions
        source: looker-source
        description: |
          The get_dimensions tool retrieves the list of dimensions defined in
          an explore.

          It takes two parameters, the model_name looked up from get_models and the
          explore_name looked up from get_explores.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "looker-get-dimensions".                                                                 |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
