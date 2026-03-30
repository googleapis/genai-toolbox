---
title: "looker-delete-project-directory"
type: docs
weight: 1
description: >
  A "looker-delete-project-directory" tool deletes a directory from a LookML project.
---

## About

A `looker-delete-project-directory` tool deletes a directory from a specified LookML project.

## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: looker-delete-project-directory
type: looker-delete-project-directory
source: looker-source
description: |
  This tool deletes a directory from a specific LookML project.
  It is useful for removing unnecessary or obsolete directories.

  Parameters:
  - project_id (string): The ID of the LookML project.
  - directory_path (string): The path of the directory to delete.

  Output:
  A string confirming the deletion of the directory.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-delete-project-directory".         |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
