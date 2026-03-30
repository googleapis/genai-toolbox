---
title: "looker-git-branch"
type: docs
weight: 1
description: >
  A "looker-git-branch" tool is used to retrieve and manipulate the git branch
  of a LookML project.
---

## About

A `looker-git-branch` tool is used to retrieve and manipulate the git branch
of a LookML project.

`looker-git-branch` requires two parameters, the LookML `project_id` and the
`operation`. The operation must be one of `list`, `get`, `create`, `switch`,
or `delete`.

The `list` operation retrieves the list of available branches. The `get`
operation retrieves the current branch.

`create`, `switch` and `delete` all require an additional parameter, the
`branch` name. The `create` operation creates a new branch. The `switch`
operation switches to the specified branch. The `delete` operation deletes
the specified branch.

`create` and `switch` can both use an additional `ref` parameter, which is
the git ref that the branch should be at. If it isn't specified, `create`
will start with the ref of the current branch. `switch` will start with the
HEAD of that branch. Specifying `ref` will do the equivalent of `reset --hard`
on the branch.

## Compatible Sources

{{< compatible-sources >}}

## Example
```yaml
kind: tool
name: project_git_branch
type: looker-git-branch
source: looker-source
description: |
  This tool is used to retrieve and manipulate the git branch of a LookML
  project.

  An operation id must be provided which is one of the following:
  * `list` - List all the available branch names.
  * `get` - Get the current branch name.
  * `create` - Create a new branch. The branch is initial set to the current
  ref, unless ref is specified. This only works in dev mode.
  * `switch` - Change the branch to the given branch, and update to the given
  ref if specified. This only works in dev mode.
  * `delete` - Delete a branch. This only works in dev mode.  

  Parameters:
  - project_id (required): The unique ID of the LookML project.
  - operation (required): One of `list`, `get`, `create`, `switch`, or `delete`.
  - branch (optional): The branch to create, switch to, or delete.
  - ref (optional): The ref to start a newly created branch, or change a branch
  with `reset --hard` on a switch operation.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-git-branch".                       |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
