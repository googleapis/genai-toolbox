---
title: "Update Project Tool"
type: docs
weight: 1
description: >
  A "looker-update-project" tool updates the configuration of an existing Looker project.
---

## About

A `looker-update-project` tool is used to update the configuration of an existing Looker project. It is crucial for configuring Git remote settings (connecting to GitHub, unsetting URL for bare mode, etc.) after a project is created.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

| **field** | **type** | **required** | **description** |
| ---------- | :------: | :----------: | ----------------------------------------- |
| project_id | string | true | The ID of the Looker project to update. |
| git_remote_url | string | false | Git remote repository URL (SSH format). |
| git_service_name | string | false | Name of the git service provider (e.g., "bare", "github"). |
| git_production_branch_name| string | false | Git production branch name. |
| pull_request_mode | string | false | The git pull request policy. Valid values: "off", "links", "recommended", "required". |
| validation_required | boolean | false | If true, validation is required before committing. |
| git_release_mgmt_enabled | boolean | false | If true, advanced git release management is enabled. |
| allow_warnings | boolean | false | If true, allow committing with warnings. |

## Example

```yaml
kind: tool
name: update_looker_project
type: looker-update-project
source: looker-source
description: |
  This tool is used to update the configuration of an existing Looker project.
```

## Reference

| **field** | **type** | **required** | **description** |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type | string | true | Must be "looker-update-project". |
| source | string | true | Name of the source. |
| description | string | true | Description of the tool that is passed to the LLM. |
