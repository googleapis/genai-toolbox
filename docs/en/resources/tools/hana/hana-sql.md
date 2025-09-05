---
title: "hana-sql"
type: docs
weight: 1
description: >
  A "hana-sql" tool executes a pre-defined SQL statement against a SAP HANA
  database.
aliases:
- /resources/tools/hana-sql
---

## About

A `hana-sql` tool executes a pre-defined SQL statement against a SAP HANA
database. It's compatible with the following source:

- [hana](../../sources/hana.md)

The specified SQL statement is executed as a parameterized query, and specified 
parameters will be inserted according to their position. If template parameters 
are included, they will be resolved before execution of the prepared statement.

## Example

> **Note:** This tool uses parameterized queries to prevent SQL injections.
> Query parameters can be used as substitutes for arbitrary expressions.
> Parameters cannot be used as substitutes for identifiers, column names, table
> names, or other parts of the query.

```yaml
tools:
 search_customers_by_region:
    kind: hana-sql
    source: my-hana-instance
    statement: |
      SELECT CUSTOMER_ID, CUSTOMER_NAME, REGION
      FROM CUSTOMERS
      WHERE REGION = ?
      AND STATUS = ?
      ORDER BY CUSTOMER_NAME
      LIMIT 50
    description: |
      Use this tool to find customers by region and status.
      Takes a region code and status and returns customer information.
      Region should be a valid region code like "EMEA", "AMERICAS", or "APAC".
      Status should be "ACTIVE" or "INACTIVE".
      Example:
      {{
          "region": "EMEA",
          "status": "ACTIVE"
      }}
    parameters:
      - name: region
        type: string
        description: Region code (e.g., EMEA, AMERICAS, APAC)
      - name: status
        type: string
        description: Customer status (ACTIVE or INACTIVE)
```

### Example with Template Parameters

> **Note:** This tool allows direct modifications to the SQL statement,
> including identifiers, column names, and table names. **This makes it more
> vulnerable to SQL injections**. Using basic parameters only (see above) is
> recommended for performance and safety reasons. For more details, please check
> [templateParameters](..#template-parameters).

```yaml
tools:
 list_table_data:
    kind: hana-sql
    source: my-hana-instance
    statement: |
      SELECT * FROM {{.schemaName}}.{{.tableName}}
      ORDER BY 1
      LIMIT 100
    description: |
      Use this tool to list data from a specific table in a schema.
      Example:
      {{
          "schemaName": "SALES",
          "tableName": "ORDERS"
      }}
    templateParameters:
      - name: schemaName
        type: string
        description: Schema name to select from
      - name: tableName
        type: string
        description: Table name to select from
```

## Reference

| **field**           |                  **type**                                 | **required** | **description**                                                                                                                            |
|---------------------|:---------------------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------------------------------------------------|
| kind                |                   string                                  |     true     | Must be "hana-sql".                                                                                                                       |
| source              |                   string                                  |     true     | Name of the source the SQL should execute on.                                                                                              |
| description         |                   string                                  |     true     | Description of the tool that is passed to the LLM.                                                                                         |
| statement           |                   string                                  |     true     | SQL statement to execute on.                                                                                                               |
| parameters          | [parameters](../#specifying-parameters)                |    false     | List of [parameters](../#specifying-parameters) that will be inserted into the SQL statement.                                           |
| templateParameters  |  [templateParameters](..#template-parameters)         |    false     | List of [templateParameters](..#template-parameters) that will be inserted into the SQL statement before executing prepared statement. |
