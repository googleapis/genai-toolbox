---
title: "looker-get-filters"
type: docs
weight: 1
description: >
  A "looker-get-filters" tool returns all the filters from a given explore
  in a given model in the source.
aliases:
- /resources/tools/looker-get-filters
---

## About

A `looker-get-filters` tool returns all the filters from a given explore
in a given model in the source.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-get-filters` accepts two parameters, the `model` and the `explore`.

## Example

```yaml
tools:
    get_filters:
        kind: looker-get-filters
        source: looker-source
        description: |
          get_filters Tool

          This tool retrieves a list of "filter-only fields" defined within a specific
          Looker explore. These are special fields defined in LookML specifically to
          create user-facing filter controls that do not directly affect the `GROUP BY`
          clause of the SQL query. They are often used in conjunction with liquid templating
          to create dynamic queries.

          Note: Regular dimensions and measures can also be used as filters in a query.
          This tool *only* returns fields explicitly defined as `filter:` in LookML.

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
| kind        |  string  |     true     | Must be "looker-get-filters".                      |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
