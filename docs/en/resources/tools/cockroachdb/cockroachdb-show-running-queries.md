---
title: cockroachdb-show-running-queries
type: docs
---

## About

The `cockroachdb-show-running-queries` tool shows all currently running queries across the CockroachDB cluster, useful for monitoring and troubleshooting.

## Configuration

```yaml
tools:
  show_running_queries:
    kind: cockroachdb-show-running-queries
    source: my-cockroachdb
    description: Shows currently running queries
```

## Parameters

This tool requires no parameters.

## Example Usage

No parameters needed - simply invoke the tool.

## Output

Returns an array of running queries with:
- `query_id`: Unique query identifier
- `txn_id`: Transaction ID
- `node_id`: Node executing the query
- `user_name`: User running the query
- `application_name`: Application identifier
- `query_start`: When the query started
- `query_running_time`: How long the query has been running (may be NULL)
- `query`: The SQL statement being executed

## Use Cases

- Identify long-running queries
- Monitor cluster activity
- Troubleshoot performance issues
- Detect blocked or stuck queries
- Audit query patterns
