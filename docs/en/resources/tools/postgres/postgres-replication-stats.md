---
title: "postgres-replication-stats"
type: docs
weight: 1
description: >
  The "postgres-replication-stats" tool reports replication-related metrics for WAL streaming replicas, including lag sizes presented in human-readable form.
aliases:
- /resources/tools/postgres-replication-stats
---

## About

The `postgres-replication-stats` tool queries pg_stat_replication to surface the status of connected replicas. It reports application_name, client address, connection and sync state, and human-readable lag sizes (sent, write, flush, replay, and total) computed using WAL LSN differences.

Compatible sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

This tool takes no parameters. It returns a JSON array; each element represents a replication connection on the primary and includes lag metrics formatted by pg_size_pretty.

## Example

```yaml
tools:
  replication_stats:
    kind: postgres-replication-stats
    source: postgres-source
    description: "Lists replication connections and readable WAL lag metrics."
```

Example response element:

```json
{
  "pid": 12345,
  "usename": "replication_user",
  "application_name": "replica-1",
  "backend_xmin": "0/0",
  "client_addr": "10.0.0.7",
  "state": "streaming",
  "sync_state": "sync",
  "sent_lag": "1234 kB",
  "write_lag": "12 kB",
  "flush_lag": "0 bytes",
  "replay_lag": "0 bytes",
  "total_lag": "1234 kB"
}
```

## Reference

| field             | type    | required | description |
|------------------:|:-------:|:--------:|:------------|
| pid               | integer | true     | Process ID of the replication backend on the primary. |
| usename           | string  | true     | Name of the user performing the replication connection. |
| application_name  | string  | true     | Name of the application (replica) connecting to the primary. |
| backend_xmin      | string  | false    | Standby's xmin horizon reported by hot_standby_feedback (may be null). |
| client_addr       | string  | false    | Client IP address of the replica (may be null). |
| state             | string  | true     | Connection state (e.g., streaming). |
| sync_state        | string  | true     | Sync state (e.g., async, sync, potential). |
| sent_lag          | string  | true     | Human-readable size difference between current WAL LSN and sent_lsn. |
| write_lag         | string  | true     | Human-readable write lag between sent_lsn and write_lsn. |
| flush_lag         | string  | true     | Human-readable flush lag between write_lsn and flush_lsn. |
| replay_lag        | string  | true     | Human-readable replay lag between flush_lsn and replay_lsn. |
| total_lag         | string  | true     | Human-readable total lag between current WAL LSN and replay_lsn. |
