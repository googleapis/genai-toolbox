---
title: "postgres-list-table-stats"
type: docs
weight: 1
description: >
  The "postgres-list-table-stats" tool reports table statistics including size, scan metrics, and bloat indicators for PostgreSQL tables.
aliases:
- /resources/tools/postgres-list-table-stats
---

## About

The `postgres-list-table-stats` tool queries `pg_stat_all_tables` to provide comprehensive statistics about tables in the database. It calculates useful metrics like index scan ratio and dead row ratio to help identify performance issues and table bloat.

Compatible sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

The tool returns a JSON array where each element represents statistics for a table, including scan metrics, row counts, and vacuum history. Results are sorted by sequential scans by default and limited to 50 rows.

## Parameters

| parameter   | type    | required | default | description |
|-------------|---------|----------|---------|-------------|
| schema_name | string  | false    | "public" | Optional: A specific schema name to filter by (supports partial matching) |
| table_name  | string  | false    | null    | Optional: A specific table name to filter by (supports partial matching) |
| owner       | string  | false    | null    | Optional: A specific owner to filter by (supports partial matching) |
| sort_by     | string  | false    | null    | Optional: The column to sort by. Valid values: `size`, `dead_rows`, `seq_scan`, `idx_scan` (defaults to `seq_scan`) |
| limit       | integer | false    | 50      | Optional: The maximum number of results to return |

## Example

```yaml
tools:
  list_table_stats:
    kind: postgres-list-table-stats
    source: postgres-source
    description: "Lists table statistics including size, scans, and bloat metrics."
```

### Example Requests

**List default tables in public schema:**
```json
{}
```

**Filter by specific table name:**
```json
{
  "table_name": "users"
}
```

**Filter by owner and sort by size:**
```json
{
  "owner": "app_user",
  "sort_by": "size",
  "limit": 10
}
```

**Find tables with high dead row ratio:**
```json
{
  "sort_by": "dead_rows",
  "limit": 20
}
```

### Example Response

```json
[
  {
    "schema_name": "public",
    "table_name": "users",
    "owner": "postgres",
    "total_size_bytes": 8388608,
    "seq_scan": 150,
    "idx_scan": 450,
    "idx_scan_ratio_percent": 75.0,
    "live_rows": 50000,
    "dead_rows": 1200,
    "dead_row_ratio_percent": 2.34,
    "n_tup_ins": 52000,
    "n_tup_upd": 12500,
    "n_tup_del": 800,
    "last_vacuum": "2025-11-27T10:30:00Z",
    "last_autovacuum": "2025-11-27T09:15:00Z",
    "last_autoanalyze": "2025-11-27T09:16:00Z"
  },
  {
    "schema_name": "public",
    "table_name": "orders",
    "owner": "postgres",
    "total_size_bytes": 16777216,
    "seq_scan": 50,
    "idx_scan": 1200,
    "idx_scan_ratio_percent": 96.0,
    "live_rows": 100000,
    "dead_rows": 5000,
    "dead_row_ratio_percent": 4.76,
    "n_tup_ins": 120000,
    "n_tup_upd": 45000,
    "n_tup_del": 15000,
    "last_vacuum": "2025-11-26T14:22:00Z",
    "last_autovacuum": "2025-11-27T02:30:00Z",
    "last_autoanalyze": "2025-11-27T02:31:00Z"
  }
]
```

## Output Fields Reference

| field                  | type      | description |
|------------------------|-----------|-------------|
| schema_name            | string    | Name of the schema containing the table. |
| table_name             | string    | Name of the table. |
| owner                  | string    | PostgreSQL user who owns the table. |
| total_size_bytes       | integer   | Total size of the table including all indexes in bytes. |
| seq_scan               | integer   | Number of sequential (full table) scans performed on this table. |
| idx_scan               | integer   | Number of index scans performed on this table. |
| idx_scan_ratio_percent | decimal   | Percentage of total scans (seq_scan + idx_scan) that used an index. A low ratio may indicate missing or ineffective indexes. |
| live_rows              | integer   | Number of live (non-deleted) rows in the table. |
| dead_rows              | integer   | Number of dead (deleted but not yet vacuumed) rows in the table. |
| dead_row_ratio_percent | decimal   | Percentage of dead rows relative to total rows. High values indicate potential table bloat. |
| n_tup_ins              | integer   | Total number of rows inserted into this table. |
| n_tup_upd              | integer   | Total number of rows updated in this table. |
| n_tup_del              | integer   | Total number of rows deleted from this table. |
| last_vacuum            | timestamp | Timestamp of the last manual VACUUM operation on this table (null if never manually vacuumed). |
| last_autovacuum        | timestamp | Timestamp of the last automatic vacuum operation on this table. |
| last_autoanalyze       | timestamp | Timestamp of the last automatic analyze operation on this table. |

## Interpretation Guide

### Index Scan Ratio (`idx_scan_ratio_percent`)

- **High ratio (> 80%)**: Table queries are efficiently using indexes. This is typically desirable.
- **Low ratio (< 20%)**: Many sequential scans indicate missing indexes or queries that cannot use existing indexes effectively. Consider adding indexes to frequently searched columns.
- **0%**: No index scans performed; all queries performed sequential scans. May warrant index investigation.

### Dead Row Ratio (`dead_row_ratio_percent`)

- **< 2%**: Healthy table with minimal bloat.
- **2-5%**: Moderate bloat; consider running VACUUM if not recent.
- **> 5%**: High bloat; may benefit from manual VACUUM or VACUUM FULL.

### Vacuum History

- **Null `last_vacuum`**: Table has never been manually vacuumed; relies on autovacuum.
- **Recent `last_autovacuum`**: Autovacuum is actively managing the table.
- **Stale timestamps**: Consider running manual VACUUM and ANALYZE if maintenance windows exist.

## Performance Considerations

- Statistics are collected from `pg_stat_all_tables`, which resets on PostgreSQL restart.
- Run `ANALYZE` on tables to update statistics for accurate query planning.
- The tool defaults to limiting results to 50 rows; adjust the `limit` parameter for larger result sets.
- Filtering by schema, table name, or owner uses `LIKE` pattern matching (supports partial matches).

## Use Cases

- **Finding ineffective indexes**: Identify tables with low `idx_scan_ratio_percent` to evaluate index strategy.
- **Detecting table bloat**: Sort by `dead_rows` to find tables needing VACUUM.
- **Monitoring growth**: Track `total_size_bytes` over time for capacity planning.
- **Audit maintenance**: Check `last_autovacuum` and `last_autoanalyze` timestamps to ensure maintenance tasks are running.
- **Understanding workload**: Examine `seq_scan` vs `idx_scan` ratios to understand query patterns.