---
title: "dataform-compile"
type: docs
weight: 1
description: > 
  A "dataform-compile" tool runs the `dataform compile` CLI command on a local project directory.
aliases:
- /resources/tools/dataform-compile
---

## About

A `dataform-compile` tool runs the `dataform compile` command on a local Dataform project.

It is a standalone tool and **is not** compatible with any sources.

`dataform-compile` takes a single required `project_dir` parameter at invocation time, which is the path to the local Dataform project directory you want to compile. The tool will execute `dataform compile --json` in that directory and return the resulting JSON object.

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
    kind: dataform-compile  
    description: Use this tool to compile a local Dataform project.
```

## Reference

| field | type | required | description |
| :---- | :---- | :---- | :---- |
| kind | string | true | Must be "dataform-compile". |
| description | string | true | Description of the tool that is passed to the LLM. |
