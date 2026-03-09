---
title: "list_tables"
type: docs
weight: 1
description: > 
   Lists all tables in the current user's schema
aliases:
- /resources/tools/oracle-sql
---

## About

An `oracle-sql` tool executes a pre-defined SQL statement against an
Oracle database. It's compatible with the following source:

- [oracle](../../sources/oracle.md)

The specified SQL statement is executed using [prepared statements][oracle-stmt]
for security and performance. It expects parameter placeholders in the SQL query
to be in the native Oracle format (e.g., `:1`, `:2`).

By default, tools are configured as **read-only** (SAFE mode). To execute data modification 
statements (INSERT, UPDATE, DELETE), you must explicitly set the `readOnly` 
field to `false`.

[oracle-stmt]: https://docs.oracle.com/javase/tutorial/jdbc/basics/prepared.html

## Example

> **Note:** This tool uses parameterized queries to prevent SQL injections.
> Query parameters can be used as substitutes for arbitrary expressions.
> Parameters cannot be used as substitutes for identifiers, column names, table
> names, or other parts of the query.

```yaml
tools:
  list_tables:
    kind: oracle-sql
    source: my-oracle-instance
    statement: |
      SELECT table_name from user_tables;
    description: |
      Lists all user tables in the connected schema, including segment size, row count, and last analyzed date. Filters by a comma-separated list of names. If names are omitted, lists all tables in the current user's schema
    