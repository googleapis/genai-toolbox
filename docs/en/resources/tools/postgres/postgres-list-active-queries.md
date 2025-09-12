---
title: "postgres-list-active-queries"
type: docs
weight: 1
description: >
  The "postgres-list-active-queries" tool lists currently active queries in a Postgres database.
aliases:
- /resources/tools/postgres-list-active-queries
---

## About

The `postgres-list-active-queries` tool retrieves information about currently active queries in a Postgres database. It's compatible with any of the following:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

`postgres-list-active-queries` lists detailed information (pid, username, database name, application name, client address, state of connection, wait event type, wait_event, backend start time, transaction start time, query start time, query duration, query text) as JSON for currently active queries. The tool takes the following input parameters:
	* `min_duraton` (optional): Only show queries running at least this long (e.g., '1 minute', '1 second', '2 seconds'). Default: '1 minute'.
	* `exclude_application_names` (optional): A comma-separated list of application names to exclude from the query results. This is useful for filtering out queries from specific applications (e.g., 'psql', 'pgAdmin', 'DBeaver'). The match is case-sensitive. Whitespace around commas and names is automatically handled. If this parameter is omitted, no applications are excluded. Default: none.
	* `limit` (optional): The maximum number of rows to return. Default: `50`.

## Example

```yaml
tools:
  list_active_queries:
    kind: postgres-list-active-queries
    source: postgres-source
    description: List the top N (default 50) currently running queries (state='active') from pg_stat_activity, ordered by longest-running first. Returns pid, user, database, application_name, client_addr, state, wait_event_type/wait_event, backend/xact/query start times, computed query_duration, and the SQL text.
```

## Reference

| **pid** | **user** | **datname** | **application_name** | **client_addr** | **state** | **wait_event_type** | **wait_event** | **backend_start** | **xact_start** | **query_start** | **query_duration** | **query** |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 446350 | postgres | postgres | psql | | active | | | 2025-09-12 12:05:25.596161+00 | 2025-09-12 12:06:28.33236+00 | 2025-09-12 12:06:28.33236+00 | 00:00:00 | SELECT pid, usename AS user, datname, application_name, client_addr, state, wait_event_type, wait_event, backend_start, xact_start, query_start, now() - query_start AS query_duration, query FROM pg_stat_activity WHERE state = 'active' ORDER BY query_duration DESC LIMIT 2; |                                         |
