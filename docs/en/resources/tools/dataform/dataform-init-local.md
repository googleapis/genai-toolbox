---
title: "dataform-init-local"
type: docs
weight: 1
description: > 
  A "dataform-init-local" tool runs the `dataform init` CLI command on a local project directory.
aliases:
- /resources/tools/dataform-init-local
---

## About

A `dataform-init-local` tool runs the `dataform init` command on a local Dataform project.

It is a standalone tool and **is not** compatible with any sources.

At invocation time, the tool executes `dataform init` in the specified project directory to set up the corresponding files for a Dataform project.

`dataform-init-local` takes the following parameter: 
- `project_dir` (string): The absolute or relative path to the local Dataform project directory. The server process must have read access to this path.

## Requirements

### Dataform CLI

This tool executes the `dataform` command-line interface (CLI) via a system call. You must have the **`dataform` CLI** installed and available in the server's system `PATH`.

You can typically install the CLI via `npm`:
```bash
npm install -g @dataform/cli
```

See the [official Dataform documentation](https://www.google.com/search?q=https://cloud.google.com/dataform/docs/install-dataform-cli) for more details.

## Example

```yaml
tools:  
  my_dataform_init:  
    kind: dataform-init-local  
    description: Use this tool to set up a local directory as a Dataform project.
```

## Reference
| **field** | **type** | **required** | **description** |
| :---- | :---- | :---- | :---- |
| kind | string | true | Must be "dataform-init-local". |
| description | string | true | Description of the tool that is passed to the LLM. |
