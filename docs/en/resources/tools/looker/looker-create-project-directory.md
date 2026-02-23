---
title: "looker-create-project-directory"
type: docs
weight: 1
description: >
  A "looker-create-project-directory" tool creates a new directory in a LookML project.
aliases:
- /resources/tools/looker-create-project-directory
---

## About

A `looker-create-project-directory` tool creates a new directory within a specified LookML project.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

## Example

```yaml
kind: tools
name: looker-create-project-directory
type: looker-create-project-directory
source: looker-source
description: |
  This tool creates a new directory within a specific LookML project.
  It is useful for organizing project files.

  Parameters:
  - project_id (string): The ID of the LookML project.
  - directory_path (string): The path of the directory to create.

  Output:
  A string confirming the creation of the directory.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-create-project-directory".         |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
