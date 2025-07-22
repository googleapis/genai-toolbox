---
title: Trino
description: Tools for working with Trino distributed SQL query engine
weight: 160
---

# Trino Tools

This section contains tools for working with [Trino](https://trino.io/), a distributed SQL query engine for big data analytics.

## Available Tools

- [trino-execute-sql](trino-execute-sql.md) - Execute arbitrary SQL queries against Trino
- [trino-sql](trino-sql.md) - Execute parameterized SQL queries against Trino

## Prerequisites

To use these tools, you need:

1. A configured [Trino source](../../sources/trino.md)
2. Proper authentication and permissions to access your Trino cluster
3. Network connectivity to the Trino coordinator

## Common Use Cases

### Data Analytics
- Query large datasets across multiple data sources
- Perform complex analytical queries with joins and aggregations
- Generate reports from federated data sources

### Data Exploration
- Explore schema and table structures across catalogs
- Sample data from different sources
- Validate data quality and consistency

### ETL Operations
- Extract data from various sources
- Transform data using SQL
- Load results into target systems

### Metadata Management
- List available catalogs, schemas, and tables
- Inspect table structures and statistics
- Analyze data distribution and properties

## Security Considerations

When using Trino tools:

1. **Access Control**: Ensure users have appropriate permissions for the catalogs and tables they need to access
2. **Network Security**: Use SSL/TLS connections for production environments
3. **Authentication**: Implement proper authentication (Basic, JWT, Kerberos)
4. **Query Limits**: Set appropriate timeouts and resource limits
5. **Audit Logging**: Enable query logging for compliance and monitoring

For more information about configuring Trino sources, see the [Trino source documentation](../../sources/trino.md).
