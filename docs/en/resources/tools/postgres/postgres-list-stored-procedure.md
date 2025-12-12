---
title: "postgres-list-stored-procedure"
type: docs
weight: 1
description: >
  The "postgres-list-stored-procedure" tool retrieves metadata for stored procedures in PostgreSQL, including procedure definitions, owners, languages, and descriptions.
aliases:
- /resources/tools/postgres-list-stored-procedure
---

## About

The `postgres-list-stored-procedure` tool queries PostgreSQL system catalogs (`pg_proc`, `pg_namespace`, `pg_roles`, and `pg_language`) to retrieve comprehensive metadata about stored procedures in the database. It filters for procedures (kind = 'p') and provides the full procedure definition along with ownership and language information.

Compatible sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

The tool returns a JSON array where each element represents a stored procedure with its schema, name, owner, language, complete definition, and optional description. Results are sorted by schema name and procedure name, with a default limit of 20 procedures.

## Parameters

| parameter    | type    | required | default | description |
|--------------|---------|----------|---------|-------------|
| role_name    | string  | false    | null    | Optional: The owner name to filter stored procedures by (supports partial matching) |
| schema_name  | string  | false    | null    | Optional: The schema name to filter stored procedures by (supports partial matching) |
| limit        | integer | false    | 20      | Optional: The maximum number of stored procedures to return |

## Example

```yaml
tools:
  list_stored_procedures:
    kind: postgres-list-stored-procedure
    source: postgres-source
    description: "Retrieves stored procedure metadata including definitions and owners."
```

### Example Requests

**List all stored procedures (default limit 20):**
```json
{}
```

**Filter by specific owner (role):**
```json
{
  "role_name": "app_user"
}
```

**Filter by schema:**
```json
{
  "schema_name": "public"
}
```

**Filter by owner and schema with custom limit:**
```json
{
  "role_name": "postgres",
  "schema_name": "public",
  "limit": 50
}
```

**Filter by partial schema name:**
```json
{
  "schema_name": "audit"
}
```

### Example Response

```json
[
  {
    "schema_name": "public",
    "name": "process_payment",
    "owner": "postgres",
    "language": "plpgsql",
    "definition": "CREATE OR REPLACE PROCEDURE public.process_payment(p_order_id integer, p_amount numeric)\n LANGUAGE plpgsql\nAS $procedure$\nBEGIN\n  UPDATE orders SET status = 'paid', amount = p_amount WHERE id = p_order_id;\n  INSERT INTO payment_log (order_id, amount, timestamp) VALUES (p_order_id, p_amount, now());\n  COMMIT;\nEND\n$procedure$",
    "description": "Processes payment for an order and logs the transaction"
  },
  {
    "schema_name": "public",
    "name": "cleanup_old_records",
    "owner": "postgres",
    "language": "plpgsql",
    "definition": "CREATE OR REPLACE PROCEDURE public.cleanup_old_records(p_days_old integer)\n LANGUAGE plpgsql\nAS $procedure$\nDECLARE\n  v_deleted integer;\nBEGIN\n  DELETE FROM audit_logs WHERE created_at < now() - (p_days_old || ' days')::interval;\n  GET DIAGNOSTICS v_deleted = ROW_COUNT;\n  RAISE NOTICE 'Deleted % records', v_deleted;\nEND\n$procedure$",
    "description": "Removes audit log records older than specified days"
  },
  {
    "schema_name": "audit",
    "name": "audit_table_changes",
    "owner": "app_user",
    "language": "plpgsql",
    "definition": "CREATE OR REPLACE PROCEDURE audit.audit_table_changes()\n LANGUAGE plpgsql\nAS $procedure$\nBEGIN\n  INSERT INTO audit.change_log (table_name, operation, changed_at) VALUES (TG_TABLE_NAME, TG_OP, now());\nEND\n$procedure$",
    "description": null
  }
]
```

## Output Fields Reference

| field       | type    | description |
|-------------|---------|-------------|
| schema_name | string  | Name of the schema containing the stored procedure. |
| name        | string  | Name of the stored procedure. |
| owner       | string  | PostgreSQL role/user who owns the stored procedure. |
| language    | string  | Programming language in which the procedure is written (e.g., plpgsql, sql, c). |
| definition  | string  | Complete SQL definition of the stored procedure, including the CREATE PROCEDURE statement. |
| description | string  | Optional description or comment for the procedure (may be null if no comment is set). |

## Use Cases

- **Code review and auditing**: Export procedure definitions for version control or compliance audits.
- **Documentation generation**: Automatically extract procedure metadata and descriptions for documentation.
- **Permission auditing**: Identify procedures owned by specific users or in specific schemas.
- **Migration planning**: Retrieve all procedure definitions when planning database migrations.
- **Dependency analysis**: Review procedure definitions to understand dependencies and call chains.
- **Security assessment**: Audit which roles own and can modify stored procedures.

## Performance Considerations

- The tool filters at the database level using LIKE pattern matching, so partial matches are supported.
- Procedure definitions can be large; consider using the `limit` parameter for large databases with many procedures.
- Results are ordered by schema name and procedure name for consistent output.
- The default limit of 20 procedures is suitable for most use cases; increase as needed.

## Notes

- Only stored **procedures** are returned; functions and other callable objects are excluded via the `prokind = 'p'` filter.
- Filtering uses `LIKE` pattern matching, so filter values support partial matches (e.g., `role_name: "app"` will match "app_user", "app_admin", etc.).
- The `definition` field contains the complete, runnable CREATE PROCEDURE statement.
- The `description` field is populated from comments set via PostgreSQL's COMMENT command and may be null.
