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


This tool identifies all locks held by active processes showing the process ID, user, query text, and an aggregated list of all transactions and specific locks (relation, mode, grant status) associated with each process.

## Query

The tool aggregates locks per backend (process) and returns the concatenated transaction ids and lock entries. The SQL used by the tool looks like:

```sql
SELECT
    locked.pid,
    locked.usename,
    locked.query,
    string_agg(locked.transactionid::text,':') as trxid,
    string_agg(locked.lockinfo,'||') as locks
FROM
    (SELECT
      a.pid,
      a.usename,
      a.query,
      l.transactionid,
      (l.granted::text||','||coalesce(l.relation::regclass,0)::text||','||l.mode::text)::text as lockinfo
    FROM
      pg_stat_activity a
      JOIN pg_locks l ON l.pid = a.pid  AND a.pid != pg_backend_pid()) as locked
GROUP BY 
    locked.pid, locked.usename, locked.query;
```

## Example

```yaml
tools:
  list_locks:
    kind: postgres-list-locks
    source: postgres-source
    description: "Lists active locks with associated process and query information."
```

Example response element (aggregated per process):

```json
{
  "pid": 23456,
  "usename": "dbuser",
  "query": "INSERT INTO orders (...) VALUES (...);",
  "trxid": "12345:0",
  "locks": "true,public.orders,RowExclusiveLock||false,0,ShareUpdateExclusiveLock"
}
```

## Reference

| field   | type    | required | description |
|:--------|:--------|:--------:|:------------|
| pid     | integer | true     | Process id (backend pid). |
| usename | string  | true     | Database user. |
| query   | string  | true     | SQL text associated with the session. |
| trxid   | string  | true     | Aggregated transaction ids for the process, joined by ':' (string). Each element is the transactionid as text. |
| locks   | string  | true     | Aggregated lock info entries for the process, joined by '||'. Each entry is a comma-separated triple: `granted,relation,mode` where `relation` may be `0` when not resolvable via regclass. |
