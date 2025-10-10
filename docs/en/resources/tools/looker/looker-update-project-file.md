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
          update_project_file Tool

          Given a project_id and a file path within the project, as well as the content
          of a LookML file, this tool will modify the file within the project.

          This tool must be called after the dev_mode tool has changed the session to
          dev mode.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "looker-update-project-file".                                                            |
| source      |                   string                   |     true     | Name of the source Looker instance.                                                              |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |