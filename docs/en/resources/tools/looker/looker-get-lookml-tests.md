---
title: "looker-get-lookml-tests"
type: docs
weight: 1
description: >
  Returns a list of tests which can be run to validate a project's LookML code and/or the underlying data, optionally filtered by the file id.
aliases:
- /resources/tools/looker-get-lookml-tests
---

## About

A "looker-get-lookml-tests" tool retrieves a list of available LookML tests for a project.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-get-lookml-tests` accepts project_id and file_id parameters.

## Example

```yaml
tools:
    get_lookml_tests:
        kind: looker-get-lookml-tests
        source: looker-source
        description: |
          Returns a list of tests which can be run to validate a project's LookML code and/or the underlying data, optionally filtered by the file id.

          Prerequisite: The Looker session must be in Development Mode. Use `dev_mode: true` first.

          Parameters:
          - project_id (required): The unique ID of the LookML project.
          - file_id (optional): The ID of the file to filter tests by. This must be the complete file path from the project root (e.g., `models/my_model.model.lkml` or `views/my_view.view.lkml`).

          Output:
          A JSON array of LookML test objects, each containing:
          - model_name: The name of the model.
          - name: The name of the test.
          - explore_name: The name of the explore being tested.
          - query_url_params: The query parameters used for the test.
          - file: The file path where the test is defined.
          - line: The line number where the test is defined.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-get-lookml-tests".                 |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
