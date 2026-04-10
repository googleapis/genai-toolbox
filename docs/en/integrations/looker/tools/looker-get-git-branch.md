---
title: "Get Git Branch Tool"
type: docs
weight: 1
description: >
  A "looker-get-git-branch" tool is used to retrieve the current git branch of a LookML project.
---

## About

A `looker-get-git-branch` tool is used to retrieve the current git branch
of a LookML project.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

| **field**  | **type** | **required** | **description**                           |
| ---------- | :------: | :----------: | ----------------------------------------- |
| project_id |  string  |     true     | The unique ID of the LookML project.      |

## Example

```yaml
kind: tool
name: get_project_git_branch
type: looker-get-git-branch
source: looker-source
description: |
  This tool is used to retrieve the current git branch of a LookML
  project.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-get-git-branch".                   |
| source      |  string  |     true     | Name of the source.                                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
