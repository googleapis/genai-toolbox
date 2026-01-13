---
title: cockroachdb-describe-table
type: docs
---

## About

The `cockroachdb-describe-table` tool retrieves detailed schema information for a specific table in CockroachDB, including column names, data types, nullability, defaults, and generated columns.

## Configuration

```yaml
tools:
  describe_table:
    kind: cockroachdb-describe-table
    source: my-cockroachdb
    description: Describes the schema of a table
```

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `schema_name` | string | Yes | The schema name (e.g., "public") |
| `table_name` | string | Yes | The table name to describe |

## Example Usage

```json
{
  "schema_name": "public",
  "table_name": "users"
}
```

## Output

Returns an array of column definitions with:
- `column_name`: Name of the column
- `data_type`: CockroachDB data type
- `is_nullable`: Whether the column allows NULL values
- `column_default`: Default value expression
- `is_hidden`: Whether the column is hidden
- `is_generated`: Generation expression for computed columns
- `ordinal_position`: Position in table definition

## Use Cases

- Explore table structure before writing queries
- Understand column types for data modeling
- Identify generated columns and defaults
- Schema migration planning
