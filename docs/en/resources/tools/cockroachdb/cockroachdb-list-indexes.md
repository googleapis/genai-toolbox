---
title: cockroachdb-list-indexes
type: docs
---

## About

The `cockroachdb-list-indexes` tool lists all indexes on a specific table in CockroachDB, including their definitions and properties.

## Configuration

```yaml
tools:
  list_indexes:
    kind: cockroachdb-list-indexes
    source: my-cockroachdb
    description: Lists all indexes on a table
```

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `schema_name` | string | Yes | The schema name (e.g., "public") |
| `table_name` | string | Yes | The table name |

## Example Usage

```json
{
  "schema_name": "public",
  "table_name": "users"
}
```

## Output

Returns an array of index information with:
- `schemaname`: Schema containing the index
- `tablename`: Table the index belongs to
- `indexname`: Name of the index
- `indexdef`: Full index definition SQL

## Use Cases

- Analyze query performance optimization opportunities
- Understand existing index coverage
- Plan index additions or removals
- Identify primary keys and unique constraints
