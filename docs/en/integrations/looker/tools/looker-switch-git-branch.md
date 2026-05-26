---
title: "Switch Git Branch Tool"
type: docs
weight: 1
description: >
  A "looker-switch-git-branch" tool is used to switch the git branch of a LookML project.
---

## About

A `looker-switch-git-branch` tool is used to switch the git branch
of a LookML project.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

| **field**  | **type** | **required** | **description**                           |
| ---------- | :------: | :----------: | ----------------------------------------- |
| project_id |  string  |     true     | The unique ID of the LookML project.      |
| branch     |  string  |     true     | The git branch to switch to.              |
| ref        |  string  |     false    | The ref to switch the branch to using `reset --hard`. |

## Example

```yaml
kind: tool
name: switch_project_git_branch
type: looker-switch-git-branch
source: looker-source
description: |
  This tool is used to switch the git branch of a LookML
  project. This only works in dev mode.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-switch-git-branch".                |
| source      |  string  |     true     | Name of the source.                                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
