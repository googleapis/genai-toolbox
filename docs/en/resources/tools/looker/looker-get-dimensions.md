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
in a given model in the source.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

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

          If this returns a suggestions field for a dimension, the contents of suggestions
          can be used as filters for this field. If this returns a suggest_explore and
          suggest_dimension, a query against that explore and dimension can be used to find
          valid filters for this field.

```

The response is a json array with the following elements:

```json
{
  "name": "field name",
  "description": "field description",
  "type": "field type",
  "label": "field label",
  "label_short": "field short label",
  "tags": ["tags", ...],
  "synonyms": ["synonyms", ...],
  "suggestions": ["suggestion", ...],
  "suggest_explore": "explore",
  "suggest_dimension": "dimension"
}
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| type        |                   string                   |     true     | Must be "looker-get-dimensions".                                                                 |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
