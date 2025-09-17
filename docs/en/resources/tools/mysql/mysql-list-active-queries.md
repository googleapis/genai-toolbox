---
title: "mysql-list-active-queries"
type: docs
weight: 1
description: >
  A "mysql-list-active-queries" tool lists active queries in a MySQL database.
aliases:
- /resources/tools/mysql-list-active-queries
---

## About

A `mysql-list-active-queries` tool retrieves information about active queries in a MySQL database. It's compatible with

- [cloud-sql-mysql](../../sources/cloud-sql-mysql.md)
- [mysql](../../sources/mysql.md)

`mysql-list-active-queries` outputs detailed information as JSON for current active queries, ordered by execution time in descending order.
This tool takes 2 optional input parameters:

- `min_duration_secs` (optional): Only show queries running for at least this long in seconds, default `0`.
- `limit` (optional):  max number of queries to return, default `10`.

## Example

```yaml
tools:
  list_active_queries:
    kind: mysql-list-active-queries
    source: my-mysql-instance
    description: Lists top N (default 10) ongoing queries from processlist and innodb_trx, ordered by execution time in descending order. Returns detailed information of those queries in json format, including process id, query, transaction duration, transaction wait duration, process time, transaction state, process state, username with host, transaction rows locked, transaction rows modified, and db schema.
```

The response is a json array with the following fields:

```json
{
  "proccess_id": "id of the MySQL process/connection this query belongs to",
  "query": "query text",
  "trx_started": "the time when the transaction (this query belongs to) started",
  "trx_duration_seconds": "the total elapsed time (in seconds) of the owning transaction so far",
  "trx_wait_duration_seconds": "the total wait time (in seconds) of the owning transaction so far",
  "query_time": "the time (in seconds) that the owning connection has been in its current state",
  "trx_state": "the transaction execution state",
  "proces_state": "the current state of the owning connection",
  "user": "the user who issued this query",
  "trx_rows_locked": "the approximate number of rows locked by the owning transaction",
  "trx_rows_modified": "the approximate number of rows modified by the owning transaction",
  "db": "the default database for the owning connection"
}
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "mysql-list-active-queries".                                                             |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
