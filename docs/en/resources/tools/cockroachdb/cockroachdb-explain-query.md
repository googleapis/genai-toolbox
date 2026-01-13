---
title: cockroachdb-explain-query
type: docs
---

## About

The `cockroachdb-explain-query` tool explains the execution plan for a SQL query in CockroachDB, helping optimize query performance.

## Configuration

```yaml
tools:
  explain_query:
    kind: cockroachdb-explain-query
    source: my-cockroachdb
    description: Explains query execution plans
```

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | The SQL query to explain |
| `verbose` | boolean | No | Include detailed execution statistics (default: false) |

## Example Usage

### Basic Explain

```json
{
  "query": "SELECT * FROM users WHERE email = 'test@example.com'"
}
```

### Verbose Explain

```json
{
  "query": "SELECT * FROM orders JOIN customers ON orders.customer_id = customers.id",
  "verbose": true
}
```

## Output

Returns the query execution plan showing:
- Query tree structure
- Index usage
- Join strategies
- Estimated row counts
- Cost estimates (with verbose mode)

## Use Cases

- Identify missing indexes
- Optimize slow queries
- Understand query execution strategies
- Troubleshoot performance issues
- Validate index effectiveness
