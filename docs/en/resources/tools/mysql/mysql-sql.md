---
title: "mysql-sql"
type: docs
weight: 1
description: >
  A "mysql-sql" tool executes a pre-defined SQL statement against a MySQL
  database.
aliases:
- /resources/tools/mysql-sql
---

## About

A `mysql-sql` tool executes a pre-defined SQL statement against a MySQL
database. It's compatible with any of the following sources:

- [cloud-sql-mysql](../../sources/cloud-sql-mysql.md)
- [mysql](../../sources/mysql.md)

The specified SQL statement is executed as a [prepared statement][mysql-prepare],
and expects parameters in the SQL query to be in the form of placeholders `?`.

[mysql-prepare]: https://dev.mysql.com/doc/refman/8.4/en/sql-prepared-statements.html

## Example

> **Note:** This tool uses parameterized queries to prevent SQL injections.
> Query parameters can be used as substitutes for arbitrary expressions.
> Parameters cannot be used as substitutes for identifiers, column names, table
> names, or other parts of the query.

```yaml
tools:
 search_flights_by_number:
    kind: mysql-sql
    source: my-mysql-instance
    statement: |
      SELECT * FROM flights
      WHERE airline = ?
      AND flight_number = ?
      LIMIT 10
    description: |
      Use this tool to get information for a specific flight.
      Takes an airline code and flight number and returns info on the flight.
      Do NOT use this tool with a flight id. Do NOT guess an airline code or flight number.
      A airline code is a code for an airline service consisting of two-character
      airline designator and followed by flight number, which is 1 to 4 digit number.
      For example, if given CY 0123, the airline is "CY", and flight_number is "123".
      Another example for this is DL 1234, the airline is "DL", and flight_number is "1234".
      If the tool returns more than one option choose the date closes to today.
      Example:
      {{
          "airline": "CY",
          "flight_number": "888",
      }}
      Example:
      {{
          "airline": "DL",
          "flight_number": "1234",
      }}
    parameters:
      - name: airline
        type: string
        description: Airline unique 2 letter identifier
      - name: flight_number
        type: string
        description: 1 to 4 digit number
```

### Example with Template Parameters

> **Note:** This tool allows direct modifications to the SQL statement,
> including identifiers, column names, and table names. **This makes it more
> vulnerable to SQL injections**. Using basic parameters only (see above) is
> recommended for performance and safety reasons. For more details, please check
> [templateParameters](..#template-parameters).

```yaml
tools:
 list_table:
    kind: mysql-sql
    source: my-mysql-instance
    statement: |
      SELECT * FROM {{.tableName}};
    description: |
      Use this tool to list all information from a specific table.
      Example:
      {{
          "tableName": "flights",
      }}
    templateParameters:
      - name: tableName
        type: string
        description: Table to select from
```

## Reference

| **field**          |                  **type**                        | **required** | **description**                                                                                                                            |
|--------------------|:------------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------------------------------------------------|
| type               |                   string                         |     true     | Must be "mysql-sql".                                                                                                                       |
| source             |                   string                         |     true     | Name of the source the SQL should execute on.                                                                                              |
| description        |                   string                         |     true     | Description of the tool that is passed to the LLM.                                                                                         |
| statement          |                   string                         |     true     | SQL statement to execute on.                                                                                                               |
| parameters         | [parameters](../#specifying-parameters)       |    false     | List of [parameters](../#specifying-parameters) that will be inserted into the SQL statement.                                           |
| templateParameters | [templateParameters](..#template-parameters) |    false     | List of [templateParameters](..#template-parameters) that will be inserted into the SQL statement before executing prepared statement. |
