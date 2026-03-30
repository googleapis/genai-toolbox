---
title: "looker-get-project-files"
type: docs
weight: 1
description: >
  A "looker-get-project-files" tool returns all the LookML fles in a project in the source.
---

## About

A `looker-get-project-files` tool returns all the lookml files in a project in the source.

`looker-get-project-files` accepts a project_id parameter.

## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: get_project_files
type: looker-get-project-files
source: looker-source
description: |
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
| type        |  string  |     true     | Must be "looker-get-project-files".                |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
