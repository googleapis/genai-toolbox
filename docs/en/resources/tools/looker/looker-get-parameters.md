---
title: "looker-get-parameters"
type: docs
weight: 1
description: >
  A "looker-get-parameters" tool returns all the parameters from a given explore
  in a given model in the source.
aliases:
- /resources/tools/looker-get-parameters
---

## About

A `looker-get-parameters` tool returns all the parameters from a given explore
in a given model in the source.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-get-parameters` accepts two parameters, the `model` and the `explore`.

## Example

```yaml
tools:
    get_parameters:
        kind: looker-get-parameters
        source: looker-source
        description: |
          The get_parameters tool retrieves the list of parameters defined in
          an explore.

          It takes two parameters, the model_name looked up from get_models and the
          explore_name looked up from get_explores.
```

The response is a json array with the following elements

```json
{
  "name": "field name",
  "description": "field description",
  "type": "field type",
  "label": "field label",
  "label_short": "field short label",
  "tags": ["tags", ...],
  "synonyms": ["synonyms", ...]
}
```


## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "looker-get-parameters".                                                                   |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
