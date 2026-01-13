---
title: cockroachdb-create-table
type: docs
---

## About

The `cockroachdb-create-table` tool creates a new table in CockroachDB using a provided CREATE TABLE statement. This is a write operation that requires write mode to be enabled.

## Configuration

```yaml
tools:
  create_table:
    kind: cockroachdb-create-table
    source: my-cockroachdb
    description: Creates a new table
```

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `table_name` | string | Yes | The name of the table to create |
| `create_statement` | string | Yes | The full CREATE TABLE SQL statement |

## Example Usage

```json
{
  "table_name": "orders",
  "create_statement": "CREATE TABLE orders (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), customer_id UUID NOT NULL, total DECIMAL(10,2), created_at TIMESTAMP DEFAULT now())"
}
```

## Output

Returns a result with:
- `table_name`: Name of the created table
- `status`: "created"
- `message`: Success message

## Security Requirements

**This tool requires write mode to be enabled:**

```yaml
sources:
  my-cockroachdb:
    readOnlyMode: false      # Disable read-only protection
    enableWriteMode: true    # Explicitly enable writes
```

If write mode is not enabled, the tool will return an error.

## Use Cases

- Create tables for new features
- Initialize database schemas
- Set up test data structures
- Provision tables for multi-tenant applications

## Best Practices

**Use UUID Primary Keys** (CockroachDB best practice):
```sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name STRING,
  email STRING UNIQUE
)
```

**Avoid Sequential IDs** to prevent transaction hotspots:
```sql
-- NOT RECOMMENDED for CockroachDB
CREATE TABLE bad_example (
  id SERIAL PRIMARY KEY  -- Sequential IDs cause hotspots
)
```

## Notes

- Requires appropriate table creation permissions
- Write operations are logged in telemetry
- Use proper CockroachDB data types (STRING instead of VARCHAR)
