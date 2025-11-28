---
title: "postgres-list-query-stats"
type: docs
weight: 1
description: >
  The "postgres-list-query-stats" tool lists query statistics from a Postgres database.
aliases:
- /resources/tools/postgres-list-query-stats
---

## About

The `postgres-list-query-stats` tool retrieves query statistics from the
`pg_stat_statements` extension in a PostgreSQL database. It provides detailed
performance metrics for executed queries. It's compatible with any of the following
sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

`postgres-list-query-stats` lists detailed query statistics as JSON, ordered by
total execution time in descending order. The tool takes the following input parameters:

- `database_name` (optional): The database name to filter query stats for. The input is
  used within a LIKE clause. Default: `""` (all databases).
- `limit` (optional): The maximum number of results to return. Default: `50`.

## Example

```yaml
tools:
  list_query_stats:
    kind: postgres-list-query-stats
    source: postgres-source
    description: List query statistics from pg_stat_statements, showing performance metrics for queries including execution counts, timing information, and resource usage. Results are ordered by total execution time descending.
```

The response is a json array with the following elements:

```json
[
  {
    "datname": "database name",
    "query": "the SQL query text",
    "calls": "number of times the query was executed",
    "total_exec_time": "total execution time in milliseconds",
    "min_exec_time": "minimum execution time in milliseconds",
    "max_exec_time": "maximum execution time in milliseconds",
    "mean_exec_time": "mean execution time in milliseconds",
    "rows": "total number of rows retrieved or affected",
    "shared_blks_hit": "number of shared block cache hits",
    "shared_blks_read": "number of shared block disk reads"
  }
]
```

## Notes

This tool requires the `pg_stat_statements` extension to be installed and enabled
on the PostgreSQL database. The `pg_stat_statements` extension tracks execution
statistics for all SQL statements executed by the server, which is useful for
identifying slow queries and understanding query performance patterns.

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| kind        |  string  |     true     | Must be "postgres-list-query-stats".                 |
| source      |  string  |     true     | Name of the source the SQL should execute on.        |
| description |  string  |     true     | Description of the tool that is passed to the LLM.   |
