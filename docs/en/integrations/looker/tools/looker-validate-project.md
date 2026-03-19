---
title: "looker-validate-project"
type: docs
weight: 1
description: >
  A "looker-validate-project" tool checks the syntax of a LookML project and reports any errors
---

## About

A "looker-validate-project" tool checks the syntax of a LookML project and reports any errors

`looker-validate-project` accepts a project_id parameter.

## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
tools:
    validate_project:
        kind: looker-validate-project
        source: looker-source
        description: |
          This tool checks a LookML project for syntax errors.

          Prerequisite: The Looker session must be in Development Mode. Use `dev_mode: true` first.

          Parameters:
          - project_id (required): The unique ID of the LookML project.

          Output:
          A list of error details including the file path and line number, and also a list of models
          that are not currently valid due to LookML errors.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-validate-project".                 |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
