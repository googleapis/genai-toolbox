---
title: "postgres-long-running-transactions"
type: docs
weight: 1
description: >
  The "postgres-long-running-transactions" tool identifies transactions that have been open longer than a configured duration and returns details about those transactions and their queries.
aliases:
- /resources/tools/postgres-long-running-transactions
---

## About

The `postgres-long-running-transactions` tool reports transactions that exceed a configured duration threshold. It scans pg_stat_activity for transactions with a non-null xact_start and computes transaction and query durations.

Compatible sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

The tool returns a JSON array with one object per matching transaction. Each object contains the process id, user, database, client address, transaction and query durations (in seconds), session state, and the SQL text currently associated with the session.

Parameters:

- `min_duration` (optional): Only show transactions running at least this long (e.g., '1 minute', '5 minutes', '30 seconds'). Default: `5 minutes`.

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
  "usename": "dbuser",
  "datname": "my_database",
  "client_addr": "10.0.0.5",
  "transaction_duration_seconds": 360.123,
  "query_duration_seconds": 120.456,
  "state": "idle in transaction",
  "query": "UPDATE users SET last_seen = now() WHERE id = 42;"
}
```

## Reference

| field                         | type    | required | description |
|------------------------------:|:-------:|:--------:|:------------|
| pid                           | integer | true     | Process id (backend pid). |
| usename                       | string  | true     | Database user name. |
| datname                       | string  | true     | Database name. |
| client_addr                   | string  | false    | Client IPv4/IPv6 address (may be null for local connections). |
| transaction_duration_seconds  | float   | true     | Seconds since xact_start. |
| query_duration_seconds        | float   | true     | Seconds since query_start. |
| state                         | string  | true     | Session state (e.g., active, idle in transaction). |
| query                         | string  | true     | SQL text associated with the session. |
