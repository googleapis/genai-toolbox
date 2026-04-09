---
title: "Delete Git Branch Tool"
type: docs
weight: 1
description: >
  A "looker-delete-git-branch" tool is used to delete a git branch of a LookML project.
---

## About

A `looker-delete-git-branch` tool is used to delete a git branch
of a LookML project.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

| **field**  | **type** | **required** | **description**                           |
| ---------- | :------: | :----------: | ----------------------------------------- |
| project_id |  string  |     true     | The unique ID of the LookML project.      |
| branch     |  string  |     true     | The git branch to delete.                 |

## Example

```yaml
kind: tool
name: delete_project_git_branch
type: looker-delete-git-branch
source: looker-source
description: |
  This tool is used to delete a git branch of a LookML
  project. This only works in dev mode.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-delete-git-branch".                |
| source      |  string  |     true     | Name of the source.                                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
