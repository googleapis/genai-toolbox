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

The `postgres-list-active-queries` tool retrieves information about currently
active queries in a Postgres database. It's compatible with any of the following
sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

`postgres-list-active-queries` lists detailed information as JSON for currently
active queries. The tool takes the following input parameters:

- `min_duraton` (optional): Only show queries running at least this long (e.g.,
  '1 minute', '1 second', '2 seconds'). Default: '1 minute'.
- `exclude_application_names` (optional): A comma-separated list of application
  names to exclude from the query results. This is useful for filtering out
  queries from specific applications (e.g., 'psql', 'pgAdmin', 'DBeaver'). The
  match is case-sensitive. Whitespace around commas and names is automatically
  handled. If this parameter is omitted, no applications are excluded.
- `limit` (optional): The maximum number of rows to return. Default: `50`.

## Example

```yaml
tools:
  list_active_queries:
    kind: postgres-list-active-queries
    source: postgres-source
    description: List the top N (default 50) currently running queries (state='active') from pg_stat_activity, ordered by longest-running first. Returns pid, user, database, application_name, client_addr, state, wait_event_type/wait_event, backend/xact/query start times, computed query_duration, and the SQL text.
```

The response is a json array with the following elements:

```json
{
  "pid": "process id",
  "user": "database user name",
  "datname": "database name",
  "application_name": "connecting application name",
  "client_addr": "connecting client ip address",
  "state": "connection state",
  "wait_event_type": "connection wait event type",
  "wait_event": "connection wait event",
  "backend_start": "connection start time",
  "xact_start": "transaction start time",
  "query_start": "query start time",
  "query_duration": "query duration",
  "query": "query text"
}
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "postgres-list-active-queries".            |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
