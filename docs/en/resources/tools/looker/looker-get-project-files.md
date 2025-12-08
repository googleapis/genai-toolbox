---
title: "looker-get-project-files"
type: docs
weight: 1
description: >
  A "looker-get-project-files" tool returns all the LookML fles in a project in the source.
aliases:
- /resources/tools/looker-get-project-files
---

## About

A `looker-get-project-files` tool returns all the lookml files in a project in the source.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-get-project-files` accepts a project_id parameter.

## Example

```yaml
tools:
    get_project_files:
        kind: looker-get-project-files
        source: looker-source
        description: |
          get_project_files Tool

          This tool retrieves a list of all LookML files within a specified project,
          providing details about each file.

          Parameters:
          - project_id (required): The unique ID of the LookML project, obtained from `get_projects`.

          Output:
          A JSON array of objects, each representing a LookML file and containing
          details such as `path`, `id`, `type`, and `git_status`.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-get-project-files".                |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
