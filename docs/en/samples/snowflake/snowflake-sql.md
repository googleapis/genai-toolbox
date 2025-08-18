---
title: "snowflake-sql"
type: docs
weight: 1
description: >
  A "snowflake-sql" tool executes a pre-defined SQL statement against a Snowflake
  database.
aliases:
- /resources/tools/snowflake-sql
---

## About

A `snowflake-sql` tool executes a pre-defined SQL statement against a Snowflake
database. It's compatible with any of the following sources:

- [snowflake](../../sources/snowflake.md)

The specified SQL statement is executed as a parameterized statement, and specified
parameters will be used according to their name: e.g. `$id`.

## Example

> **Note:** This tool uses parameterized queries to prevent SQL injections.
> Query parameters can be used as substitutes for arbitrary expressions.
> Parameters cannot be used as substitutes for identifiers, column names, table
> names, or other parts of the query.

```yaml
tools:
    list_tables:
        kind: snowflake-sql
        source: snowflake-source
        description: "Lists detailed schema information (object type, columns, constraints, indexes, owner, comment) as JSON for user-created tables. Filters by a comma-separated list of names. If names are omitted, lists all tables in the specified database and schema."
        statement: |
           WITH
           input_param AS (
                SELECT ? AS param -- Single bind variable here
           )
           ,
           all_tables_mode AS (
                SELECT COALESCE(TRIM(param), '') = '' AS is_all_tables
                FROM input_param
           ) --SELECT * FROM all_tables_mode;
           ,
           filtered_table_names AS (
                SELECT DISTINCT TRIM(LOWER(value)) AS table_name
                FROM input_param, all_tables_mode, TABLE(SPLIT_TO_TABLE(param, ','))
                WHERE NOT is_all_tables
           ) -- SELECT * FROM filtered_table_names;
           ,
           table_info AS (
                SELECT
                    t.TABLE_CATALOG,
                    t.TABLE_SCHEMA,
                    t.TABLE_NAME,
                    t.TABLE_TYPE,
                    t.TABLE_OWNER,
                    t.COMMENT
                FROM
                    all_tables_mode
                    CROSS JOIN ${SNOWFLAKE_DATABASE}.INFORMATION_SCHEMA.TABLES T
                WHERE
                    t.TABLE_TYPE = 'BASE TABLE'
                    AND t.TABLE_SCHEMA NOT IN ('INFORMATION_SCHEMA')
                    AND t.TABLE_SCHEMA = '${SNOWFLAKE_SCHEMA}'
                    AND is_all_tables OR LOWER(T.TABLE_NAME) IN (SELECT table_name FROM filtered_table_names)
            ) -- SELECT * FROM table_info;
            ,
            columns_info AS (
                SELECT
                    c.TABLE_CATALOG AS database_name,
                    c.TABLE_SCHEMA AS schema_name,
                    c.TABLE_NAME AS table_name,
                    c.COLUMN_NAME AS column_name,
                    c.DATA_TYPE AS data_type,
                    c.ORDINAL_POSITION AS column_ordinal_position,
                    c.IS_NULLABLE AS is_nullable,
                    c.COLUMN_DEFAULT AS column_default,
                    c.COMMENT AS column_comment
                FROM
                    ${SNOWFLAKE_DATABASE}.INFORMATION_SCHEMA.COLUMNS c
                    INNER JOIN table_info USING (TABLE_CATALOG, TABLE_SCHEMA, TABLE_NAME)
            )
            ,
            constraints_info AS (
                SELECT
                    tc.TABLE_CATALOG AS database_name,
                    tc.TABLE_SCHEMA AS schema_name,
                    tc.TABLE_NAME AS table_name,
                    tc.CONSTRAINT_NAME AS constraint_name,
                    tc.CONSTRAINT_TYPE AS constraint_type
                FROM
                    ${SNOWFLAKE_DATABASE}.INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
                    INNER JOIN table_info USING (TABLE_CATALOG, TABLE_SCHEMA, TABLE_NAME)
                GROUP BY
                    tc.TABLE_CATALOG, tc.TABLE_SCHEMA, tc.TABLE_NAME, tc.CONSTRAINT_NAME, tc.CONSTRAINT_TYPE
            )
            SELECT
                ti.TABLE_SCHEMA AS schema_name,
                ti.TABLE_NAME AS object_name,
                OBJECT_CONSTRUCT(
                    'schema_name', ti.TABLE_SCHEMA,
                    'object_name', ti.TABLE_NAME,
                    'object_type', ti.TABLE_TYPE,
                    'owner', ti.TABLE_OWNER,
                    'comment', ti.COMMENT,
                    'columns', COALESCE(
                        (SELECT ARRAY_AGG(
                            OBJECT_CONSTRUCT(
                                'column_name', ci.column_name,
                                'data_type', ci.data_type,
                                'ordinal_position', ci.column_ordinal_position,
                                'is_nullable', ci.is_nullable,
                                'column_default', ci.column_default,
                                'column_comment', ci.column_comment
                            )
                        ) FROM columns_info ci WHERE ci.table_name = ti.TABLE_NAME AND ci.schema_name = ti.TABLE_SCHEMA),
                        ARRAY_CONSTRUCT()
                    ),
                    'constraints', COALESCE(
                        (SELECT ARRAY_AGG(
                            OBJECT_CONSTRUCT(
                                'constraint_name', cons.constraint_name,
                                'constraint_type', cons.constraint_type
                            )
                        ) FROM constraints_info cons WHERE cons.table_name = ti.TABLE_NAME AND cons.schema_name = ti.TABLE_SCHEMA),
                        ARRAY_CONSTRUCT()
                    )
                ) AS object_details
            FROM table_info ti
            ORDER BY ti.TABLE_SCHEMA, ti.TABLE_NAME;
        parameters:
            - name: table_names
              type: string
              description: "Optional: A comma-separated list of table names. If empty, details for all tables in the specified database and schema will be listed."
```

## Reference

| **field**          |                  **type**                        | **required** | **description**                                                                                                                            |
|--------------------|:------------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------------------------------------------------|
| kind               |                   string                         |     true     | Must be "snowflake-sql".                                                                                                                   |
| source             |                   string                         |     true     | Name of the source the SQL query should execute on.                                                                                        |
| description        |                   string                         |     true     | Description of the tool that is passed to the LLM.                                                                                         |
| statement          |                   string                         |     true     | SQL statement to execute                                                                                                                   |
| parameters         | [parameters](../#specifying-parameters)       |    false     | List of [parameters](../#specifying-parameters) that will be used with the SQL statement.                                               |
| templateParameters | [templateParameters](#template-parameters) |    false     | List of [templateParameters](#template-parameters) that will be inserted into the SQL statement before executing prepared statement. |
| authRequired       |                array[string]                     |    false     | List of auth services that are required to use this tool.                                                                                  |