---
title: "looker-get-project-directories"
type: docs
weight: 1
description: >
  A "looker-get-project-directories" tool returns the directories within a specific LookML project.
aliases:
- /resources/tools/looker-get-project-directories
---

## About

A `looker-get-project-directories` tool retrieves the directories within a specified LookML project.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

## Example

```yaml
kind: tools
name: looker-get-project-directories
type: looker-get-project-directories
source: looker-source
description: |
  This tool retrieves a list of directories within a specific LookML project.
  It is useful for exploring the project structure.

  Parameters:
  - project_id (string): The ID of the LookML project.

  Output:
  A JSON array of strings, representing the directories within the project.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-get-project-directories".          |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
