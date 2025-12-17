---
title: "postgres-get-column-cardinality"
type: docs
weight: 1
description: >
  The "postgres-get-column-cardinality" tool estimates the number of unique values in one or all columns of a Postgres database table.
aliases:
- /resources/tools/postgres-get-column-cardinality
---

## About

The `postgres-get-column-cardinality` tool estimates the number of unique values
(cardinality) for one or all columns in a specific PostgreSQL table by using the
database's internal statistics. It's compatible with any of the following sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

`postgres-get-column-cardinality` returns detailed information as JSON about column
cardinality values, ordered by estimated cardinality in descending order. The tool takes
the following input parameters:

- `schema_name` (required): The schema name in which the table is present.
- `table_name` (required): The table name in which the column is present.
- `column_name` (optional): The column name for which the cardinality is to be found.
  If not provided, cardinality for all columns will be returned. Default: `""`.

## Example

```yaml
tools:
  get_column_cardinality:
    kind: postgres-get-column-cardinality
    source: postgres-source
    description: Estimates the number of unique values (cardinality) quickly for one or all columns in a specific PostgreSQL table by using the database's internal statistics, returning the results in descending order of estimated cardinality. Please run ANALYZE on the table before using this tool to get accurate results. The tool returns the column_name and the estimated_cardinality. If the column_name is not provided, the tool returns all columns along with their estimated cardinality.
```

The response is a json array with the following elements:

```json
[
  {
    "column_name": "name of the column",
    "estimated_cardinality": "estimated number of unique values in the column"
  }
]
```

## Notes

For accurate results, it's recommended to run `ANALYZE` on the table before using this
tool. The `ANALYZE` command updates the database statistics that this tool relies on
to estimate cardinality.

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| kind        |  string  |     true     | Must be "postgres-get-column-cardinality".           |
| source      |  string  |     true     | Name of the source the SQL should execute on.        |
| description |  string  |     true     | Description of the tool that is passed to the LLM.   |
