---
title: "postgres-long-running-transactions"
type: docs
weight: 1
description: >
  The postgres-long-running-transactions tool Identifies and lists database transactions that exceed a specified time limit. For each of the long running transactions, the output contains the process id, database name, user name, application name, client address, state, connection age, transaction age, query age, last activity age, wait event type, wait event, and query string.
aliases:
- /resources/tools/postgres-long-running-transactions
---

## About

The `postgres-long-running-transactions` tool reports transactions that exceed a configured duration threshold by scanning `pg_stat_activity` for sessions where `xact_start` is set and older than the configured interval.

Compatible sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

The tool returns a JSON array with one object per matching session (non-idle). Each object contains the process id, database and user, application name, client address, session state, several age intervals (connection, transaction, query, and last activity), wait event info, and the SQL text currently associated with the session.

Parameters:

- `min_duration` (optional): Only show transactions running at least this long (Postgres interval format, e.g., '5 minutes'). Default: `5 minutes`.
- `limit` (optional): Maximum number of results to return. Default: `20`.

## Query

The SQL used by the tool looks like:

```sql
SELECT
  pid,
  datname,
  usename,
  application_name as appname,
  client_addr,
  state,
  now() - backend_start as conn_age,
  now() - xact_start as xact_age,
  now() - query_start as query_age,
  now() - state_change as last_activity_age,
  wait_event_type,
  wait_event,
  query
FROM
  pg_stat_activity
WHERE
  state <> 'idle'
  AND (now() - xact_start) > COALESCE($1::INTERVAL, interval '5 minutes')
  AND xact_start IS NOT NULL
  AND pid <> pg_backend_pid()
ORDER BY
  xact_age DESC
LIMIT 
  COALESCE($2::int, 20);
```

## Example

```yaml
tools:
  long_running_transactions:
    kind: postgres-long-running-transactions
    source: postgres-source
    description: "Identifies transactions open longer than a threshold and returns details including query text and durations."
```

Example response element:

```json
{
  "pid": 12345,
  "datname": "my_database",
  "usename": "dbuser",
  "appname": "my_app",
  "client_addr": "10.0.0.5",
  "state": "idle in transaction",
  "conn_age": "00:12:34",
  "xact_age": "00:06:00",
  "query_age": "00:02:00",
  "last_activity_age": "00:01:30",
  "wait_event_type": null,
  "wait_event": null,
  "query": "UPDATE users SET last_seen = now() WHERE id = 42;"
}
```

## Reference

| field                | type    | required | description |
|:---------------------|:--------|:--------:|:------------|
| pid                  | integer | true     | Process id (backend pid). |
| datname              | string  | true     | Database name. |
| usename              | string  | true     | Database user name. |
| appname              | string  | false    | Application name (client application). |
| client_addr          | string  | false    | Client IPv4/IPv6 address (may be null for local connections). |
| state                | string  | true     | Session state (e.g., active, idle in transaction). |
| conn_age             | string  | true     | Age of the connection: `now() - backend_start` (Postgres interval serialized as string). |
| xact_age             | string  | true     | Age of the transaction: `now() - xact_start` (Postgres interval serialized as string). |
| query_age            | string  | true     | Age of the currently running query: `now() - query_start` (Postgres interval serialized as string). |
| last_activity_age    | string  | true     | Time since last state change: `now() - state_change` (Postgres interval serialized as string). |
| wait_event_type      | string  | false    | Type of event the backend is waiting on (may be null). |
| wait_event           | string  | false    | Specific wait event name (may be null). |
| query                | string  | true     | SQL text associated with the session. |
