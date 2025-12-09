---
title: "postgres-list-pg-settings"
type: docs
weight: 1
description: >
 The "postgres-list-pg-settings" tool lists PostgreSQL run-time configuration settings.
aliases:
- /resources/tools/postgres-list-pg-settings
---

## About

The `postgres-list-pg-settings` tool lists the configuration parameters for the postgres server, their current values, and related information. It's compatible with any of the following sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

`postgres-list-pg-settings` lists detailed information as JSON for each setting. The tool
takes the following input parameters:

- `setting_name` (optional): A text to filter results by setting name. Default: `""`
- `limit` (optional): The maximum number of rows to return. Default: `50`.

## Example

```yaml
tools:
  list_indexes:
    kind: postgres-list-pg-settings
    source: postgres-source
    description: |
        Lists configuration parameters for the postgres server ordered lexicographically, 
        with a default limit of 50 rows. It returns the parameter name, its current setting,
        unit of measurement, a short description, the source of the current setting (e.g.,
        default, configuration file, session), and whether a restart is required when the
        parameter value is changed."
```

The response is a json array with the following elements:

```json
{
 "name": "Setting name",
 "current_value": "Current value of the setting",
 "unit": "Unit of the setting",
 "short_desc": "Short description of the setting",
 "source": "Source of the current value (e.g., default, configuration file, session)",
 "requires_restart": "Indicates if a server restart is required to apply a change ('Yes', 'No', or 'No (Reload sufficient)')"
}
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| kind        |  string  |     true     | Must be "postgres-list-pg-settings".                     |
| source      |  string  |     true     | Name of the source the SQL should execute on.        |
| description |  string  |    false     | Description of the tool that is passed to the agent. |
