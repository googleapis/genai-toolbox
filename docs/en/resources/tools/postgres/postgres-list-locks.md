---
title: "postgres-list-locks"
type: docs
weight: 1
description: >
  The "postgres-list-locks" tool lists active locks in the database, including the associated process, lock type, relation, mode, and the query holding or waiting on the lock.
aliases:
- /resources/tools/postgres-list-locks
---

## About

The `postgres-list-locks` tool displays information about active locks by joining pg_stat_activity with pg_locks. This is useful to find transactions holding or waiting for locks and to troubleshoot contention.

Compatible sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

Parameters:

- `only_current_database` (optional): If `true`, lists active locks for the current database only. If `false` (default), lists locks across all databases.

The tool returns a JSON array; each element includes pid, user, database name, client address, query start/age, lock type, table name, lock mode, whether the lock is granted, and the query text.

## Example

```yaml
tools:
  list_locks:
    kind: postgres-list-locks
    source: postgres-source
    description: "Lists active locks with associated process and query information."
```

Example response element:

```json
{
  "pid": 23456,
  "usename": "dbuser",
  "datname": "my_database",
  "client_addr": "10.0.0.6",
  "query_start": "2025-10-21T12:34:56Z",
  "query_age": "00:02:34",
  "state": "active",
  "locktype": "relation",
  "table_name": "public.orders",
  "mode": "RowExclusiveLock",
  "granted": true,
  "query": "INSERT INTO orders (...) VALUES (...);"
}
```

## Reference

| field        | type    | required | description |
|-------------:|:-------:|:--------:|:------------|
| pid          | integer | true     | Process id (backend pid). |
| usename      | string  | true     | Database user. |
| datname      | string  | true     | Database name. |
| client_addr  | string  | false    | Client IP address (may be null). |
| query_start  | string  | true     | Timestamp when the current query started. |
| query_age    | string  | true     | Human-readable age of the query (age(now(), query_start)). |
| state        | string  | true     | Session state. |
| locktype     | string  | true     | Type of lock (e.g., relation, tuple). |
| table_name   | string  | false    | Relation name (resolved with regclass when available). |
| mode         | string  | true     | Lock mode (e.g., AccessShareLock, RowExclusiveLock). |
| granted      | boolean | true    | Whether the lock is granted (true) or waiting (false). |
| query        | string  | true     | SQL text associated with the session. |
