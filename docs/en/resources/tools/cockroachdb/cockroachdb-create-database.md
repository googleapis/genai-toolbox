---
title: cockroachdb-create-database
type: docs
---

## About

The `cockroachdb-create-database` tool creates a new database in CockroachDB. This is a write operation that requires write mode to be enabled.

## Configuration

```yaml
tools:
  create_database:
    kind: cockroachdb-create-database
    source: my-cockroachdb
    description: Creates a new database
```

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `database_name` | string | Yes | The name of the database to create |

## Example Usage

```json
{
  "database_name": "analytics"
}
```

## Output

Returns a result with:
- `database_name`: Name of the created database
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

- Initialize new application environments
- Create databases for multi-tenant applications
- Set up test or staging databases
- Provision databases for new services

## Notes

- Uses `CREATE DATABASE IF NOT EXISTS` to avoid errors if database exists
- Requires appropriate database permissions
- Write operations are logged in telemetry
