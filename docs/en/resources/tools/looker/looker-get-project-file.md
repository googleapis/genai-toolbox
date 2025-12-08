---
title: "looker-get-project-file"
type: docs
weight: 1
description: >
  A "looker-get-project-file" tool returns the contents of a LookML fle.
aliases:
- /resources/tools/looker-get-project-file
---

## About

A `looker-get-project-file` tool returns the contents of a LookML file.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-get-project-file` accepts a project_id parameter and a file_path parameter.

## Example

```yaml
tools:
    get_project_file:
        kind: looker-get-project-file
        source: looker-source
        description: |
          get_project_file Tool

          This tool retrieves the raw content of a specific LookML file from within a project.

          Parameters:
          - project_id (required): The unique ID of the LookML project, obtained from `get_projects`.
          - file_path (required): The path to the LookML file within the project,
            typically obtained from `get_project_files`.

          Output:
          The raw text content of the specified LookML file.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-get-project-file".                 |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
