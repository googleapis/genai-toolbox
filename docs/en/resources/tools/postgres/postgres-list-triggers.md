---
title: "postgres-list-triggers"
type: docs
weight: 1
description: >
 The "postgres-list-triggers" tool lists triggers in a Postgres database.
aliases:
- /resources/tools/postgres-list-triggers
---

## About

The `postgres-list-triggers` tool lists available non-internal triggers in the database. It's compatible with any of the following sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

`postgres-list-triggers` lists detailed information as JSON for triggers. The tool takes the following input parameters:

- `trigger_name` (optional): A text to filter results by trigger name. The input is used within a LIKE clause. Default: `""`
- `schema_name` (optional): A text to filter results by schema name. The input is used within a LIKE clause. Default: `""`
- `table_name` (optional): A text to filter results by table name. The input is used within a LIKE clause. Default: `""`
- `limit` (optional): The maximum number of triggers to return. Default: `50`


## Example

```yaml
```yaml
tools:
  list_triggers:
    kind: postgres-list-triggers
    source: postgres-source
    description: |
        Lists all non-internal triggers in a database. Returns trigger name, schema name, table name, wether its enabled or disabled,  timing (e.g BEFORE/AFTER of the event), the  events that cause the trigger to fire such as INSERT, UPDATE, or DELETE, whether the trigger activates per ROW or per STATEMENT, the handler function executed by the trigger and full definition.
```

The response is a json array with the following elements:
```json
{
 "trigger_name": "trigger name",
 "schema_name": "schema name",
 "table_name": "table name",
 "status": "Whether the trigger is currently active (ENABLED, DISABLED, REPLICA, ALWAYS).",
 "timing": "When it runs relative to the event (BEFORE, AFTER, INSTEAD OF).",
 "events": "The specific operations that fire it (INSERT, UPDATE, DELETE, TRUNCATE)",
 "activation_level": "Granularity of execution (ROW vs STATEMENT).",
 "function_name": "The function it executes",
 "definition": "Full SQL definition of the trigger"
}
```

## Reference

| **field**   | **type** | **required**  | **description**                                      |
|-------------|:--------:|:-------------:|------------------------------------------------------|
| kind        |  string  |     true      | Must be "postgres-list-triggers".                      |
| source      |  string  |     true      | Name of the source the SQL should execute on.        |
| description |  string  |     false     | Description of the tool that is passed to the agent. |
