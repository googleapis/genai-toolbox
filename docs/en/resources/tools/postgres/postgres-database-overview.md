---
title: "postgres-database-overview"
type: docs
weight: 1
description: >
  The "postgres-database-overview" fetches the current state of the PostgreSQL server. 
aliases:
- /resources/tools/postgres-database-overview
---

## About

The `postgres-database-overview` fetches the current state of the PostgreSQL
server. It's compatible with any of the following sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

`postgres-database-overview` fetches the current state of the PostgreSQL server
This tool does not take any input parameters.

## Example

```yaml
tools:
  database_overview:
    kind: postgres-database-overview
    source: cloudsql-pg-source
    description: |
    fetches the current state of the PostgreSQL server. It returns the postgres version, whether it's a replica, uptime duration, maximum connection limit, number of current connections, number of active connections and the percentage of connections in use.
```

The response is a JSON object with the following elements:

```json
{
 "pg_version": "PostgreSQL server version string",
 "is_replica": "boolean indicating if the instance is in recovery mode",
 "uptime": "interval string representing the total server uptime",
 "max_connections": "integer maximum number of allowed connections",
 "current_connections": "integer number of current connections",
 "active_connections": "integer number of currently active connections",
 "pct_connections_used": "float percentage of max_connections currently in use"
}
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| kind        |  string  |     true     | Must be "postgres-database-overview".                |
| source      |  string  |     true     | Name of the source the SQL should execute on.        |
| description |  string  |    false     | Description of the tool that is passed to the agent. |
