---
title: "postgres-list-database-stats"
type: docs
weight: 1
description: >
 The "postgres-list-database-stats" tool lists lists key performance and activity statistics of PostgreSQL databases.
aliases:
- /resources/tools/postgres-list-database-stats
---

## About

The `postgres-list-database-stats` lists the key performance and activity statistics for each PostgreSQL database in the instance, offering insights into cache efficiency, transaction throughput, row-level activity, temporary file usage, and contention. It's compatible with
any of the following sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

`postgres-list-database-stats` lists detailed information as JSON for each database. The tool
takes the following input parameters:

- `database_name` (optional): A text to filter results by database name. Default: `""`
- `include_templates` (optional): Boolean, set to `true` to include template databases in the results. Default: `false`
- `database_owner` (optional): A text to filter results by database owner.  Default: `""`
- `default_tablespace` (optional): A text to filter results by the default tablespace name. Default: `""`
- `order_by` (optional): Specifies the sorting order. Valid values are `'size'` (descending) or `'commit'` (descending). Default: `database_name` ascending.
- `limit` (optional): The maximum number of databases to return. Default: `10`

## Example

```yaml
tools:
  list_database_stats:
    kind: postgres-list-database-stats
    source: postgres-source
    description: |
       Lists the key performance and activity statistics for each PostgreSQL
        database in the instance, offering insights into cache efficiency,
        transaction throughput row-level activity, temporary file usage, and
        contention. It returns: the database name, whether the database is
        connectable, database owner, default tablespace name, the percentage of
        data blocks found in the buffer cache rather than being read from disk
        (a higher value indicates better cache performance), the total number of
        disk blocks read from disk, the total number of times disk blocks were
        found already in the cache; the total number of committed transactions,
        the total number of rolled back transactions, the percentage of rolled
        back transactions compared to the total number of completed
        transactions, the total number of rows returned by queries, the total
        number of live rows fetched by scans, the total number of rows inserted,
        the total number of rows updated, the total number of rows deleted, the
        number of temporary files created by queries, the total size of
        temporary files used by queries in bytes, the number of query
        cancellations due to conflicts with recovery, the number of deadlocks
        detected, the current number of active backend connections, the
        timestamp when the database statistics were last reset, and the total
        database size in bytes.
```

The response is a json array with the following elements:

```json
{
 "database_name": "Name of the database",
 "is_connectable": "Boolean indicating Whether the database allows connections",
 "database_owner": "Username of the database owner",
 "default_tablespace": "Name of the default tablespace for the database",
 "cache_hit_ratio_percent": "The percentage of data blocks found in the buffer cache rather than being read from disk",
 "blocks_read_from_disk": "The total number of disk blocks read for this database",
 "blocks_hit_in_cache": "The total number of times disk blocks were found already in the cache.",
 "xact_commit": "The total number of committed transactions",
 "xact_rollback": "The total number of rolled back transactions",
 "rollback_ratio_percent": "The percentage of rolled back transactions compared to the total number of completed transactions",
 "rows_returned_by_queries": "The total number of rows returned by queries",
 "rows_fetched_by_scans": "The total number of live rows fetched by scans",
 "tup_inserted": "The total number of rows inserted",
 "tup_updated": "The total number of rows updated",
 "tup_deleted": "The total number of rows deleted",
 "temp_files": "The number of temporary files created by queries",
 "temp_size_bytes": "The total size of temporary files used by queries in bytes",
 "conflicts": "Number of query cancellations due to conflicts",
 "deadlocks": "Number of deadlocks detected",
 "active_connections": "The current number of active backend connections",
 "statistics_last_reset": "The timestamp when the database statistics were last reset",
 "database_size_bytes": "The total disk size of the database in bytes"
}
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| kind        |  string  |     true     | Must be "postgres-list-database-stats".              |
| source      |  string  |     true     | Name of the source the SQL should execute on.        |
| description |  string  |    false     | Description of the tool that is passed to the agent. |
