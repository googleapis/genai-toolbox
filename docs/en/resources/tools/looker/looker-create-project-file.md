---
title: "looker-create-project-file"
type: docs
weight: 1
description: >
  A "looker-create-project-file" tool creates a new LookML file in a project.
aliases:
- /resources/tools/looker-create-project-file
---

## About

A `looker-create-project-file` tool creates a new LookML file in a project

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-create-project-file` accepts a project_id parameter and a file_path parameter
as well as the file content.

## Example

```yaml
tools:
    create_project_file:
        kind: looker-create-project-file
        source: looker-source
        description: |
          This tool creates a new LookML file within a specified project, populating
          it with the provided content.

          Prerequisite: The Looker session must be in Development Mode. Use `dev_mode: true` first.

          Parameters:
          - project_id (required): The unique ID of the LookML project.
          - file_path (required): The desired path and filename for the new file within the project.
          - content (required): The full LookML content to write into the new file.

          Output:
          A confirmation message upon successful file creation.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-create-project-file".              |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
