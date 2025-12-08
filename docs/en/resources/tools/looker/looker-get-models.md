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

A `looker-get-models` tool returns all the models in the source.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-get-models` accepts no parameters.

## Example

```yaml
tools:
    get_models:
        kind: looker-get-models
        source: looker-source
        description: |
          get_models Tool

          This tool retrieves a list of available LookML models in the Looker instance.
          LookML models define the data structure and relationships that users can query.
          The output includes details like the model's `name` and `label`, which are
          essential for subsequent calls to tools like `get_explores` or `query`.

          This tool takes no parameters.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-get-models".                       |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
