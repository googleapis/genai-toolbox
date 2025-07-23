---
title: trino-execute-sql
description: Execute arbitrary SQL queries against Trino
---

# trino-execute-sql

Execute arbitrary SQL queries against a Trino cluster. This tool provides dynamic SQL execution capabilities for ad-hoc queries, data exploration, and analysis.

## Configuration

```yaml
tools:
  my-trino-executor:
    kind: trino-execute-sql
    source: my-trino-source
    description: Execute SQL queries against Trino
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `kind` | string | Yes | Must be `trino-execute-sql` |
| `source` | string | Yes | Name of the Trino source to use |
| `description` | string | Yes | Description of what this tool does |
| `authRequired` | array | No | List of required authentication scopes |

## Usage

The tool accepts a single parameter:

### Input Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `sql` | string | Yes | The SQL query to execute |

### Example Queries

#### Basic SELECT Query
```json
{
  "sql": "SELECT * FROM hive.sales.orders LIMIT 10"
}
```

#### Cross-Catalog Query
```json
{
  "sql": "SELECT o.order_id, c.customer_name FROM hive.sales.orders o JOIN postgresql.crm.customers c ON o.customer_id = c.id WHERE o.order_date >= DATE '2024-01-01'"
}
```

#### Aggregation Query
```json
{
  "sql": "SELECT region, COUNT(*) as order_count, SUM(total_amount) as total_revenue FROM hive.sales.orders WHERE order_date >= DATE '2024-01-01' GROUP BY region ORDER BY total_revenue DESC"
}
```

#### Data Definition Query
```json
{
  "sql": "CREATE TABLE hive.analytics.daily_sales AS SELECT DATE(order_date) as sale_date, SUM(total_amount) as daily_revenue FROM hive.sales.orders GROUP BY DATE(order_date)"
}
```

## Response Format

The tool returns results as an array of objects, where each object represents a row:

```json
[
  {
    "order_id": "12345",
    "customer_name": "John Doe",
    "total_amount": 99.99,
    "order_date": "2024-01-15"
  },
  {
    "order_id": "12346",
    "customer_name": "Jane Smith",
    "total_amount": 149.99,
    "order_date": "2024-01-16"
  }
]
```

## Supported Operations

### Query Operations
- `SELECT` - Retrieve data from tables
- `WITH` - Common table expressions
- `UNION`, `INTERSECT`, `EXCEPT` - Set operations
- `JOIN` - Various join types across catalogs

### Data Definition Language (DDL)
- `CREATE TABLE` - Create new tables
- `CREATE VIEW` - Create views
- `ALTER TABLE` - Modify table structure
- `DROP TABLE`, `DROP VIEW` - Remove objects

### Data Manipulation Language (DML)
- `INSERT` - Add new rows
- `UPDATE` - Modify existing rows (if supported by connector)
- `DELETE` - Remove rows (if supported by connector)

### Trino-Specific Operations
- `SHOW CATALOGS` - List available catalogs
- `SHOW SCHEMAS` - List schemas in a catalog
- `SHOW TABLES` - List tables in a schema
- `DESCRIBE` - Show table structure
- `SHOW STATS` - Display table statistics
- `ANALYZE TABLE` - Compute table statistics

## Example Configurations

### Basic Configuration
```yaml
sources:
  trino-cluster:
    kind: trino
    host: trino.example.com
    port: "8080"
    user: analyst
    catalog: hive
    schema: default

tools:
  execute-query:
    kind: trino-execute-sql
    source: trino-cluster
    description: Execute ad-hoc SQL queries against Trino
```

### With Authentication
```yaml
sources:
  secure-trino:
    kind: trino
    host: trino.example.com
    port: "8443"
    user: analyst
    password: secret
    catalog: hive
    schema: analytics
    sslEnabled: true

tools:
  secure-query:
    kind: trino-execute-sql
    source: secure-trino
    description: Execute queries on secure Trino cluster
    authRequired:
      - trino.query
```

## Best Practices

### Performance
1. **Use LIMIT**: Add LIMIT clauses to prevent accidentally returning huge result sets
2. **Filter Early**: Apply WHERE clauses to reduce data scanning
3. **Use Appropriate Data Types**: Ensure proper data type usage in queries
4. **Leverage Pushdown**: Structure queries to take advantage of connector pushdown capabilities

### Security
1. **Validate Inputs**: Always validate SQL queries before execution
2. **Use Least Privilege**: Grant minimal necessary permissions
3. **Audit Queries**: Log all executed queries for security monitoring
4. **Sanitize Outputs**: Be careful with sensitive data in results

### Error Handling
1. **Check Syntax**: Validate SQL syntax before execution
2. **Handle Timeouts**: Set appropriate query timeouts
3. **Graceful Failures**: Handle connection and execution errors properly

## Common Use Cases

### Data Analysis
```json
{
  "sql": "SELECT product_category, AVG(rating) as avg_rating, COUNT(*) as review_count FROM hive.reviews.product_reviews WHERE review_date >= DATE '2024-01-01' GROUP BY product_category HAVING COUNT(*) >= 100 ORDER BY avg_rating DESC"
}
```

### Data Quality Checks
```json
{
  "sql": "SELECT 'null_emails' as check_name, COUNT(*) as failure_count FROM postgresql.users.customers WHERE email IS NULL UNION ALL SELECT 'duplicate_emails' as check_name, COUNT(*) - COUNT(DISTINCT email) as failure_count FROM postgresql.users.customers"
}
```

### Cross-System Data Validation
```json
{
  "sql": "SELECT 'order_mismatch' as check_name, ABS(h.hive_count - p.postgres_count) as difference FROM (SELECT COUNT(*) as hive_count FROM hive.sales.orders WHERE order_date = DATE '2024-01-15') h CROSS JOIN (SELECT COUNT(*) as postgres_count FROM postgresql.orders.daily_orders WHERE order_date = DATE '2024-01-15') p"
}
```

## Error Handling

Common errors and their meanings:

- **Syntax Error**: Invalid SQL syntax
- **Table Not Found**: Referenced table doesn't exist or user lacks permissions
- **Column Not Found**: Referenced column doesn't exist in the table
- **Timeout**: Query execution exceeded the configured timeout
- **Permission Denied**: User lacks necessary permissions for the operation
- **Catalog/Schema Not Found**: Specified catalog or schema doesn't exist

## Compatible Sources

- [`trino`](../../sources/trino.md) - Trino distributed SQL query engine

## Related Tools

- [`trino-sql`](trino-sql.md) - Execute parameterized SQL queries with predefined statements
