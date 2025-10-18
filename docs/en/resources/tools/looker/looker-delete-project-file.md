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
          delete_project_file Tool

          Given a project_id and a file path within the project, this tool will delete
          the file from the project.

          This tool must be called after the dev_mode tool has changed the session to
          dev mode.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "looker-delete-project-file".                                                            |
| source      |                   string                   |     true     | Name of the source Looker instance.                                                              |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |