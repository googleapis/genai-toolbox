---
title: "Create Git Branch Tool"
type: docs
weight: 1
description: >
  A "looker-create-git-branch" tool is used to create a new git branch for a LookML project.
---

## About

A `looker-create-git-branch` tool is used to create a new git branch
for a LookML project.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

| **field**  | **type** | **required** | **description**                           |
| ---------- | :------: | :----------: | ----------------------------------------- |
| project_id |  string  |     true     | The unique ID of the LookML project.      |
| branch     |  string  |     true     | The git branch to create.                 |
| ref        |  string  |     false    | The ref to use as the start of a new branch. Defaults to HEAD of current branch. |

## Example

```yaml
kind: tool
name: create_project_git_branch
type: looker-create-git-branch
source: looker-source
description: |
  This tool is used to create a new git branch of a LookML
  project. This only works in dev mode.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-create-git-branch".                |
| source      |  string  |     true     | Name of the source.                                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
