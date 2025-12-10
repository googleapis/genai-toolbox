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
          create_project_file Tool

          Given a project_id and a file path within the project, as well as the content
          of a LookML file, this tool will create a new file within the project.

          This tool must be called after the dev_mode tool has changed the session to
          dev mode.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-create-project-file".              |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
