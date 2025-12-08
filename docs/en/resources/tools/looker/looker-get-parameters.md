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
          get_parameters Tool

          This tool retrieves a list of parameters defined within a specific Looker explore.
          LookML parameters are dynamic input fields that allow users to influence query
          behavior without directly modifying the underlying LookML. They are often used
          with `liquid` templating to create flexible dashboards and reports, enabling
          users to choose dimensions, measures, or other query components at runtime.

          Parameters:
          - model_name (required): The name of the LookML model, obtained from `get_models`.
          - explore_name (required): The name of the explore within the model, obtained from `get_explores`.
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

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-get-parameters".                   |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
