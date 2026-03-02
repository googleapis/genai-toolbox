---
title: "looker-run-lookml-tests"
type: docs
weight: 1
description: >
  This tool runs LookML tests in the project, filtered by file, test, and/or model.
aliases:
- /resources/tools/looker-run-lookml-tests
---

## About

A "looker-run-lookml-tests" tool executes specific LookML tests within a project.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-run-lookml-tests` accepts project_id, file_id, test, and model parameters.

## Example

```yaml
tools:
    run_lookml_tests:
        kind: looker-run-lookml-tests
        source: looker-source
        description: |
          This tool runs LookML tests in the project, filtered by file, test, and/or model. These filters work in conjunction (logical AND).

          Prerequisite: The Looker session must be in Development Mode. Use `dev_mode: true` first.

          Parameters:
          - project_id (required): The unique ID of the project to run LookML tests for.
          - file_id (optional): The ID of the file to run tests for. This must be the complete file path from the project root (e.g., `models/my_model.model.lkml` or `views/my_view.view.lkml`).
          - test (optional): The name of the test to run.
          - model (optional): The name of the model to run tests for.

          Output:
          A JSON array containing the results of the executed tests, where each object includes:
          - model_name: Name of the model tested.
          - test_name: Name of the test.
          - assertions_count: Total number of assertions in the test.
          - assertions_failed: Number of assertions that failed.
          - success: Boolean indicating if the test passed.
          - errors: Array of error objects (if any), containing details like `message`, `file_path`, `line_number`, and `severity`.
          - warnings: Array of warning messages (if any).
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-run-lookml-tests".                 |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
