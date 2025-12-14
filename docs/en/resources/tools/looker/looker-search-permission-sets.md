---
title: "looker-search-permission-sets"
type: docs
weight: 1
description: >
  A "looker-search-permission-sets" tool searches for permission sets in the Looker instance.
aliases:
- /resources/tools/looker-search-permission-sets
---

## About

A `looker-search-permission-sets` tool searches for permission sets in the Looker instance. Permission sets define what actions a user or group can perform.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-search-permission-sets` accepts the following parameters:
- `name` (optional): The name of the permission set.
- `id` (optional): The unique id of the permission set.
- `limit` (optional): The number of results to return.
- `offset` (optional): The number of results to skip before returning.

## Example

```yaml
tools:
    search_permission_sets:
        kind: looker-search-permission-sets
        source: looker-source
        description: |
          This tool searches for permission sets in the Looker instance.
          Permission sets define what actions a user or group can perform.
          The output includes details like the permission set's `id`, `name`, `permissions`, and `all_access`.

          Parameters:
          - name (optional): The name of the permission set.
          - id (optional): The unique id of the permission set.
          - limit (optional): The number of results to return.
          - offset (optional): The number of results to skip before returning.

          Output:
          A JSON array of objects, where each object represents a permission set and contains details
          such as `id`, `name`, `permissions` (list of permission strings), `all_access` (boolean), and `built_in` (boolean).
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-search-permission-sets".           |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
