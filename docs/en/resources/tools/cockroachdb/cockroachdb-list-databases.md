---
title: cockroachdb-list-databases
type: docs
---

## About

The `cockroachdb-list-databases` tool lists all databases in the CockroachDB cluster.

## Configuration

```yaml
tools:
  list_databases:
    kind: cockroachdb-list-databases
    source: my-cockroachdb
    description: List all databases in the cluster
```

## Parameters

This tool requires no parameters.

## Example Usage

No parameters needed - simply invoke the tool.

## Output

Returns an array of databases with:
- `database_name`: Name of the database

## Use Cases

- List all available databases in the cluster
- Database inventory management
- Identify databases for operations
