---
title: "trino-analyze"
type: docs
weight: 4
description: >
  A "trino-analyze" tool analyzes SQL queries to provide execution plans,
  performance insights, and optimization recommendations for Trino.
aliases:
- /resources/tools/trino-analyze
---

## About

A `trino-analyze` tool provides query analysis and optimization capabilities
for Trino SQL queries. It can generate execution plans in various formats,
validate query syntax, analyze query performance, and provide optimization
recommendations.

This tool is compatible with:
- [trino](../../sources/trino.md)

## Features

- **Query Plan Generation**: Generate execution plans in text, JSON, or
  graphviz formats
- **Performance Analysis**: Run ANALYZE to get actual execution statistics
- **Query Validation**: Validate SQL syntax without executing the query
- **Distributed Plans**: View distributed execution plans showing how work is
  distributed across the cluster
- **Optimization Recommendations**: Get suggestions for query optimization
  based on the execution plan
- **Performance Warnings**: Identify potential performance issues like cross
  joins or full table scans

## Example

```yaml
tools:
  analyze_query:
    kind: trino-analyze
    source: my-trino-instance
    description: Analyze SQL queries for performance insights and optimization
```

## Parameters

| **parameter**  | **type** | **required** | **description**                                                                                              |
|----------------|:--------:|:------------:|--------------------------------------------------------------------------------------------------------------|
| query          | string   | true         | The SQL query to analyze. Can be SELECT, INSERT, UPDATE, or DELETE.                                         |
| format         | string   | false        | Output format: 'text' (default), 'json', 'graphviz', or 'summary'.                                          |
| analyze        | boolean  | false        | If true, runs ANALYZE to get actual execution statistics (may execute query). Default: false.               |
| distributed    | boolean  | false        | If true, shows distributed execution plan. Default: true.                                                   |
| validate       | boolean  | false        | If true, only validates query syntax without generating a plan. Default: false.                             |

## Response Structure

The tool returns a JSON object with the following structure:

```json
{
  "query": "SELECT * FROM table WHERE id = 1",
  "plan": "Fragment 0 [SOURCE]...",
  "planJson": {...},  // When format is 'json'
  "statistics": {
    "totalCpuTime": "1.2s",
    "totalScheduledTime": "1.5s",
    "rawInputRows": 1000000,
    "processedRows": 1000,
    "outputRows": 1
  },
  "recommendations": [
    "Consider adding indexes to avoid full table scans",
    "Hash join detected - ensure sufficient memory is available"
  ],
  "warnings": [
    "Query estimated to process 1000000 rows - consider adding filters or limits"
  ],
  "estimatedCost": 1250.5,
  "estimatedRows": 1000,
  "isValid": true,
  "validationErrors": []
}
```

## Usage Examples

### Basic Query Analysis
```yaml
# Get a text execution plan
parameters:
  query: "SELECT * FROM orders WHERE order_date > '2024-01-01'"
  format: "text"
```

### Performance Analysis with Statistics
```yaml
# Run ANALYZE to get actual execution statistics
parameters:
  query: "SELECT customer_id, COUNT(*) FROM orders GROUP BY customer_id"
  format: "json"
  analyze: true
```

### Query Validation Only
```yaml
# Validate SQL syntax without generating a plan
parameters:
  query: "SELECT * FROM non_existent_table"
  validate: true
```

### Simplified Summary
```yaml
# Get a simplified explanation of the query plan
parameters:
  query: "SELECT o.*, c.* FROM orders o JOIN customers c ON o.customer_id = c.id"
  format: "summary"
```

## Reference

| **field**      | **type** | **required** | **description**                                                |
|----------------|:--------:|:------------:|----------------------------------------------------------------|
| kind           | string   | true         | Must be "trino-analyze".                                      |
| source         | string   | true         | Name of the Trino source to connect to.                       |
| description    | string   | true         | Description of the tool that is passed to the LLM.            |
