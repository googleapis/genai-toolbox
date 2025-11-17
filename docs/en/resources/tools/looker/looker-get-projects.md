---
title: "looker-get-projects"
type: docs
weight: 1
description: >
  A "looker-get-projects" tool returns all the LookML projects in the source.
aliases:
- /resources/tools/looker-get-projects
---

## About

A `looker-get-projects` tool returns all the projects in the source.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-get-projects` accepts no parameters.

## Example

```yaml
tools:
    get_projects:
        kind: looker-get-projects
        source: looker-source
        description: |
          get_projects Tool

          This tool returns the project_id and project_name for
          all the LookML projects on the looker instance.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-get-projects".                     |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
