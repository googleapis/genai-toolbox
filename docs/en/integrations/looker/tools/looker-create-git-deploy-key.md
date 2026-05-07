---
title: "looker-create-git-deploy-key Tool"
type: docs
weight: 1
description: >
  A "looker-create-git-deploy-key" tool generates a public SSH deploy key for a Looker project to connect to a remote Git repository.
---

## About

A `looker-create-git-deploy-key` tool generates a public/private key pair for authenticating SSH Git requests from Looker to a remote Git repository for a specific project. It returns the public deploy key string, which must be added to the remote repository (e.g., GitHub) with write permissions.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

| **field** | **type** | **required** | **description** |
| ---------- | :------: | :----------: | ----------------------------------------- |
| project_id | string | true | The ID of the Looker project. |

## Example

```yaml
kind: tool
name: create_project_deploy_key
type: looker-create-git-deploy-key
source: looker-source
description: |
  This tool is used to generate a public SSH deploy key for a Looker project.
```

## Reference

| **field** | **type** | **required** | **description** |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type | string | true | Must be "looker-create-git-deploy-key". |
| source | string | true | Name of the source. |
| description | string | true | Description of the tool that is passed to the LLM. |
