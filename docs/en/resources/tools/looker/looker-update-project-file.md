---
title: "looker-update-project-file"
type: docs
weight: 1
description: >
  A "looker-update-project-file" tool updates the content of a LookML file in a project.
aliases:
- /resources/tools/looker-update-project-file
---

## About

A `looker-update-project-file` tool updates the content of a LookML file.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-update-project-file` accepts a project_id parameter and a file_path parameter
as well as the new file content.

## Example

```yaml
tools:
    update_project_file:
        kind: looker-update-project-file
        source: looker-source
        description: |
          This tool modifies the content of an existing LookML file within a specified project.

          Prerequisite: The Looker session must be in Development Mode. Use `dev_mode: true` first.

          Parameters:
          - project_id (required): The unique ID of the LookML project.
          - file_path (required): The exact path to the LookML file to modify within the project.
          - content (required): The new, complete LookML content to overwrite the existing file.

          Output:
          A confirmation message upon successful file modification.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-update-project-file".              |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
