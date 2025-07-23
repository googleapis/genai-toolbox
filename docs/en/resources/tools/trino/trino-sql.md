---
title: trino-sql
description: Execute parameterized SQL queries against Trino
---

# trino-sql

Execute parameterized SQL queries against a Trino cluster. This tool allows you to define reusable SQL queries with parameters, providing a safer and more structured approach compared to dynamic SQL execution.

## Configuration

```yaml
tools:
  get-sales-by-region:
    kind: trino-sql
    source: my-trino-source
    description: Get sales data for a specific region and date range
    statement: |
      SELECT 
        order_id,
        customer_name,
        total_amount,
        order_date
      FROM hive.sales.orders 
      WHERE region = $1 
        AND order_date >= $2 
        AND order_date <= $3
      ORDER BY order_date DESC
    parameters:
      - name: region
        type: string
        description: The region to filter by
        required: true
      - name: start_date
        type: string
        description: Start date (YYYY-MM-DD format)
        required: true
      - name: end_date
        type: string
        description: End date (YYYY-MM-DD format)
        required: true
```

### Configuration Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `kind` | string | Yes | Must be `trino-sql` |
| `source` | string | Yes | Name of the Trino source to use |
| `description` | string | Yes | Description of what this tool does |
| `statement` | string | Yes | The SQL query with parameter placeholders ($1, $2, etc.) |
| `parameters` | array | No | List of parameter definitions |
| `templateParameters` | array | No | Parameters for template substitution |
| `authRequired` | array | No | List of required authentication scopes |

### Parameter Definition

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Parameter name |
| `type` | string | Yes | Parameter type (string, number, boolean) |
| `description` | string | Yes | Description of the parameter |
| `required` | boolean | No | Whether the parameter is required (default: false) |

## Parameter Placeholders

Use numbered placeholders in your SQL statements:
- `$1` - First parameter
- `$2` - Second parameter
- `$3` - Third parameter
- And so on...

Parameters are passed to the query in the order they are defined in the `parameters` array.

## Usage Examples

### Basic Query with Parameters

```yaml
tools:
  get-customer-orders:
    kind: trino-sql
    source: trino-cluster
    description: Get orders for a specific customer
    statement: |
      SELECT 
        order_id,
        product_name,
        quantity,
        price,
        order_date
      FROM hive.sales.orders o
      JOIN hive.catalog.products p ON o.product_id = p.id
      WHERE o.customer_id = $1
      ORDER BY o.order_date DESC
      LIMIT $2
    parameters:
      - name: customer_id
        type: string
        description: The customer ID
        required: true
      - name: limit
        type: number
        description: Maximum number of orders to return
        required: false
```

**Input:**
```json
{
  "customer_id": "CUST123",
  "limit": 10
}
```

### Cross-Catalog Query

```yaml
tools:
  customer-analytics:
    kind: trino-sql
    source: trino-cluster
    description: Analyze customer behavior across systems
    statement: |
      SELECT 
        c.customer_id,
        c.customer_name,
        c.registration_date,
        COALESCE(o.order_count, 0) as total_orders,
        COALESCE(o.total_spent, 0) as total_spent
      FROM postgresql.crm.customers c
      LEFT JOIN (
        SELECT 
          customer_id,
          COUNT(*) as order_count,
          SUM(total_amount) as total_spent
        FROM hive.sales.orders
        WHERE order_date >= $1
        GROUP BY customer_id
      ) o ON c.customer_id = o.customer_id
      WHERE c.registration_date >= $2
      ORDER BY total_spent DESC
    parameters:
      - name: order_date_start
        type: string
        description: Start date for order analysis (YYYY-MM-DD)
        required: true
      - name: customer_registration_start
        type: string
        description: Start date for customer registration (YYYY-MM-DD)
        required: true
```

### Aggregation with Optional Filters

```yaml
tools:
  sales-summary:
    kind: trino-sql
    source: trino-cluster
    description: Generate sales summary with optional region filter
    statement: |
      SELECT 
        DATE_TRUNC('month', order_date) as month,
        region,
        COUNT(*) as order_count,
        SUM(total_amount) as total_revenue,
        AVG(total_amount) as avg_order_value
      FROM hive.sales.orders
      WHERE order_date >= $1
        AND order_date <= $2
        AND (NULLIF($3, '') IS NULL OR region = $3)
      GROUP BY DATE_TRUNC('month', order_date), region
      ORDER BY month DESC, total_revenue DESC
    parameters:
      - name: start_date
        type: string
        description: Start date (YYYY-MM-DD)
        required: true
      - name: end_date
        type: string
        description: End date (YYYY-MM-DD)
        required: true
      - name: region
        type: string
        description: Optional region filter (leave empty for all regions)
        required: false
```

## Response Format

The tool returns results as an array of objects, where each object represents a row:

```json
[
  {
    "order_id": "ORD-001",
    "customer_name": "John Doe",
    "total_amount": 299.99,
    "order_date": "2024-01-15"
  },
  {
    "order_id": "ORD-002",
    "customer_name": "Jane Smith",
    "total_amount": 149.99,
    "order_date": "2024-01-14"
  }
]
```

## Template Parameters

For advanced use cases, you can use template parameters to dynamically construct parts of the query:

```yaml
tools:
  flexible-query:
    kind: trino-sql
    source: trino-cluster
    description: Query with dynamic table selection
    statement: |
      SELECT *
      FROM {{table_name}}
      WHERE created_date >= $1
      LIMIT $2
    templateParameters:
      - name: table_name
        type: string
        description: The table to query
        required: true
    parameters:
      - name: start_date
        type: string
        description: Start date filter
        required: true
      - name: limit
        type: number
        description: Maximum rows to return
        required: true
```

## Best Practices

### Security
1. **Use Parameters**: Always use parameterized queries to prevent SQL injection
2. **Validate Inputs**: Validate parameter values before query execution
3. **Least Privilege**: Grant minimal necessary permissions to the Trino user
4. **Audit Logging**: Log parameter values for security monitoring

### Performance
1. **Efficient Filtering**: Place selective filters early in WHERE clauses
2. **Use LIMIT**: Add LIMIT clauses to prevent large result sets
3. **Index Awareness**: Structure queries to leverage available indexes
4. **Pushdown Optimization**: Write queries that can push filters to source systems

### Maintainability
1. **Clear Descriptions**: Write descriptive parameter descriptions
2. **Logical Grouping**: Group related parameters together
3. **Consistent Naming**: Use consistent parameter naming conventions
4. **Documentation**: Document complex query logic

## Common Patterns

### Optional Filters

Handle optional parameters using NULLIF and OR conditions:

```sql
WHERE (NULLIF($1, '') IS NULL OR column1 = $1)
  AND (NULLIF($2, '') IS NULL OR column2 = $2)
```

### Date Range Queries

```sql
WHERE date_column >= DATE($1)
  AND date_column <= DATE($2)
```

### Dynamic LIMIT

```sql
LIMIT COALESCE(NULLIF($1, 0), 100)  -- Default to 100 if parameter is 0 or null
```

### IN Clauses

For multiple values, you can use string splitting:

```sql
WHERE column_name IN (
  SELECT value 
  FROM UNNEST(SPLIT($1, ',')) AS t(value)
)
```

## Error Handling

Common parameter-related errors:

- **Missing Required Parameter**: A required parameter was not provided
- **Invalid Parameter Type**: Parameter type doesn't match expected type
- **Invalid Date Format**: Date parameters must be in YYYY-MM-DD format
- **Parameter Out of Range**: Numeric parameters exceed valid ranges
- **SQL Syntax Error**: Error in the prepared statement

## Compatible Sources

- [`trino`](../../sources/trino.md) - Trino distributed SQL query engine

## Related Tools

- [`trino-execute-sql`](trino-execute-sql.md) - Execute arbitrary SQL queries without predefined parameters
