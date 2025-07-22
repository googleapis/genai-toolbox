---
title: "looker-get-measures"
type: docs
weight: 1
description: > 
  A "looker-get-measures" tool returns all the measures from a given explore
  in a given model in the source.
aliases:
- /resources/tools/looker-get-measures
---

## About

A `looker-get-measures` tool returns all the measures from a given explore
in a given mode in the source.

It's compatible with the following sources:

- [looker](../sources/looker.md)

`looker-get-measures` accepts two parameters, the `model` and the `explore`.

## Example

```yaml
tools:
    get_measures:
        kind: looker-get-measures
        source: looker-source
        description: |
          The get_measures tool retrieves the list of measures defined in
          an explore.

          It takes two parameters, the model_name looked up from get_models and the
          explore_name looked up from get_explores.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "looker-get-measures".                                                                   |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
