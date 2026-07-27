---
title: "looker-create-project Tool"
type: docs
weight: 1
description: >
  A "looker-create-project" tool is used to create a new Looker project.
---

## About

A `looker-create-project` tool is used to create a new Looker project. This is the first step in setting up a LookML project. It starts as a bare repository by default.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

| **field** | **type** | **required** | **description** |
| ---------- | :------: | :----------: | ----------------------------------------- |
| name | string | true | The unique name of the new project. |
| git_production_branch_name | string | false | Git production branch name. Defaults to master. |
| pull_request_mode | string | false | The git pull request policy. Valid values: "off", "links", "recommended", "required". |
| validation_required | boolean | false | If true, validation is required before committing. |
| git_release_mgmt_enabled | boolean | false | If true, advanced git release management is enabled. |
| allow_warnings | boolean | false | If true, allow committing with warnings when validation is required. |

## Example

```yaml
kind: tool
name: create_looker_project
type: looker-create-project
source: looker-source
description: |
  This tool is used to create a new Looker project.
```

## Reference

| **field** | **type** | **required** | **description** |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type | string | true | Must be "looker-create-project". |
| source | string | true | Name of the source. |
| description | string | true | Description of the tool that is passed to the LLM. |
