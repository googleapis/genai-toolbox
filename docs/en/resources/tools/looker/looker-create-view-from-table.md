---
title: "looker-create-view-from-table"
type: docs
weight: 1
description: >
  This tool generates boilerplate LookML views directly from the database schema.
aliases:
- /resources/tools/looker-create-view-from-table
---

## About

A "looker-create-view-from-table" tool triggers the automatic generation of LookML view files based on database tables.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-create-view-from-table` accepts project_id, connection, tables, and folder_name parameters.

## Example

```yaml
tools:
    create_view_from_table:
        kind: looker-create-view-from-table
        source: looker-source
        description: |
          This tool generates boilerplate LookML views directly from the database schema.
          It does not create model or explore files, only view files in the specified folder.

          Prerequisite: The Looker session must be in Development Mode. Use `dev_mode: true` first.

          Parameters:
          - project_id (required): The unique ID of the LookML project.
          - connection (required): The database connection name.
          - tables (required): A list of objects to generate views for. Each object must contain `schema` and `table_name` (note: table names are case-sensitive). Optional fields include `primary_key`, `base_view`, and `columns` (array of objects with `column_name`).
          - folder_name (optional): The folder to place the view files in (defaults to 'views/').

          Output:
          A confirmation message upon successful view generation, or an error message if the operation fails.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-create-view-from-table".           |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
