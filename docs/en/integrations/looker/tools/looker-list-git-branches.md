---
title: "List Git Branches Tool"
type: docs
weight: 1
description: >
  A "looker-list-git-branches" tool is used to retrieve the list of available git branches of a LookML project.
---

## About

A `looker-list-git-branches` tool is used to retrieve the list of available git branches
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
name: list_project_git_branches
type: looker-list-git-branches
source: looker-source
description: |
  This tool is used to retrieve the list of available git branches of a LookML
  project.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-list-git-branches".                |
| source      |  string  |     true     | Name of the source.                                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
