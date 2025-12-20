---
title: "looker-delete-project-file"
type: docs
weight: 1
description: >
  A "looker-delete-project-file" tool deletes a LookML file in a project.
aliases:
- /resources/tools/looker-delete-project-file
---

## About

A `looker-delete-project-file` tool deletes a LookML file in a project

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-delete-project-file` accepts a project_id parameter and a file_path parameter.

## Example

```yaml
tools:
    delete_project_file:
        kind: looker-delete-project-file
        source: looker-source
        description: |
          This tool permanently deletes a specified LookML file from within a project.
          Use with caution, as this action cannot be undone through the API.

          Prerequisite: The Looker session must be in Development Mode. Use `dev_mode: true` first.

          Parameters:
          - project_id (required): The unique ID of the LookML project.
          - file_path (required): The exact path to the LookML file to delete within the project.

          Output:
          A confirmation message upon successful file deletion.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-delete-project-file".              |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
