---
title: "clickhouse-sql"
type: docs
weight: 2
description: >
  A "clickhouse-sql" tool executes SQL queries as prepared statements in ClickHouse.
aliases:
- /resources/tools/clickhouse-sql
---

## About

A `clickhouse-sql` tool executes SQL queries as prepared statements against a 
ClickHouse database. It's compatible with the [clickhouse](../../sources/clickhouse.md) source.

This tool supports both template parameters (for SQL statement customization) 
and regular parameters (for prepared statement values), providing flexible 
query execution capabilities.

## Example

```yaml
tools:
  my_analytics_query:
    kind: clickhouse-sql
    source: my-clickhouse-instance
    description: Get user analytics for a specific date range
    statement: |
      SELECT 
        user_id,
        count(*) as event_count,
        max(timestamp) as last_event
      FROM events 
      WHERE date >= ? AND date <= ?
      GROUP BY user_id
      ORDER BY event_count DESC
      LIMIT ?
    parameters:
      - name: start_date
        description: Start date for the query (YYYY-MM-DD format)
      - name: end_date  
        description: End date for the query (YYYY-MM-DD format)
      - name: limit
        description: Maximum number of results to return
```

## Template Parameters Example

```yaml
tools:
  flexible_table_query:
    kind: clickhouse-sql
    source: my-clickhouse-instance
    description: Query any table with flexible columns
    statement: |
      SELECT {{columns}}
      FROM {{table_name}}
      WHERE created_date >= ?
      LIMIT ?
    templateParameters:
      - name: columns
        description: Comma-separated list of columns to select
      - name: table_name
        description: Name of the table to query
    parameters:
      - name: start_date
        description: Start date filter
      - name: limit
        description: Maximum number of results
```

## Reference

| **field**          | **type**           | **required** | **description**                                           |
|--------------------|:------------------:|:------------:|-----------------------------------------------------------|
| type               | string             | true         | Must be "clickhouse-sql".                                |
| source             | string             | true         | Name of the ClickHouse source to execute SQL against.    |
| description        | string             | true         | Description of the tool that is passed to the LLM.       |
| statement          | string             | true         | The SQL statement template to execute.                    |
| parameters         | array of Parameter | false        | Parameters for prepared statement values.                 |
| templateParameters | array of Parameter | false        | Parameters for SQL statement template customization.     |