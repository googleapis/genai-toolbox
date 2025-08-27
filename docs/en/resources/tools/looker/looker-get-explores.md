---
title: "looker-get-explores"
type: docs
weight: 1
description: >
  A "looker-get-explores" tool returns all explores
  for the given model from the source.
aliases:
- /resources/tools/looker-get-explores
---

## About

A `looker-get-explores` tool returns all explores
for a given model from the source.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-get-explores` accepts one parameter, the
`model` id.

The return type is an array of maps, each map is formatted like:

```json
{
    "name": "explore name",
    "description": "explore description",
    "label": "explore label",
    "group_label": "group label"
```
## Example

```yaml
tools:
    get_explores:
        kind: looker-get-explores
        source: looker-source
        description: |
          The get_explores tool retrieves the list of explores defined in a LookML model
          in the Looker system.

          It takes one parameter, the model_name looked up from get_models.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "looker-get-explores".                                                                   |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
