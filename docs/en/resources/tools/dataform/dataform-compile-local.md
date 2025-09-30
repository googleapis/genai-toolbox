---
title: "dataform-compile-local"
type: docs
weight: 1
description: > 
  A "dataform-compile-local" tool runs the `dataform compile` CLI command on a local project directory.
aliases:
- /resources/tools/dataform-compile-local
---

## About

A `dataform-compile-local` tool runs the `dataform compile` command on a local Dataform project.

It is a standalone tool and **is not** compatible with any sources.

At invocation time, the tool executes `dataform compile --json` in the specified project directory and returns the resulting JSON object from the CLI.

`dataform-compile-local` takes the following parameter: 
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
  my_dataform_compiler:  
    kind: dataform-compile-local  
    description: Use this tool to compile a local Dataform project.
```

## Reference
| field | type | required | description |
| :---- | :---- | :---- | :---- |
| kind | string | true | Must be "dataform-compile-local". |
| description | string | true | Description of the tool that is passed to the LLM. |
