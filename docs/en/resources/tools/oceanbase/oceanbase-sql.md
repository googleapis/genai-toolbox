---
title: "oceanbase-sql"
type: docs
weight: 1
description: > 
  An "oceanbase-sql" tool executes a pre-defined SQL statement against an OceanBase database.
aliases:
- /resources/tools/oceanbase-sql
---

## About

An `oceanbase-sql` tool executes a pre-defined SQL statement against an
OceanBase database. It's compatible with the following source:

- [oceanbase](../../sources/oceanbase.md)

The specified SQL statement is executed as a [prepared
statement][mysql-prepare], and expects parameters in the SQL query to be in the
form of placeholders `?`.

[mysql-prepare]: https://dev.mysql.com/doc/refman/8.4/en/sql-prepared-statements.html

## Example

> **Note:** This tool uses parameterized queries to prevent SQL injections.
> Query parameters can be used as substitutes for arbitrary expressions.
> Parameters cannot be used as substitutes for identifiers, column names, table
> names, or other parts of the query.

```yaml
tools:
  search_flights_by_number:
    kind: oceanbase-sql
    source: my-oceanbase-instance
    statement: |
      SELECT * FROM flights
      WHERE airline = ?
      AND flight_number = ?
      LIMIT 10
    description: |
      Use this tool to get information for a specific flight.
      Takes an airline code and flight number and returns info on the flight.
      Do NOT use this tool with a flight id. Do NOT guess an airline code or flight number.
      Example:
      {{
          "airline": "CY",
          "flight_number": "888",
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
> recommended for performance and safety reasons.

```yaml
tools:
  list_table:
    kind: oceanbase-sql
    source: my-oceanbase-instance
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

### Example with Array Parameters

```yaml
tools:
  search_flights_by_ids:
    kind: oceanbase-sql
    source: my-oceanbase-instance
    statement: |
      SELECT * FROM flights
      WHERE id IN (?)
      AND status IN (?)
    description: |
      Use this tool to get information for multiple flights by their IDs and statuses.
      Example:
      {{
          "flight_ids": [1, 2, 3],
          "statuses": ["active", "scheduled"]
      }}
    parameters:
      - name: flight_ids
        type: array
        description: List of flight IDs to search for
        items:
          name: flight_id
          type: integer
          description: Individual flight ID
      - name: statuses
        type: array
        description: List of flight statuses to filter by
        items:
          name: status
          type: string
          description: Individual flight status
```

## Reference

| **field**          |                     **type**                     | **required** | **description**                                                                                                                            |
|--------------------|:------------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------------------------------------------------|
| kind               |                      string                      |     true     | Must be "oceanbase-sql".                                                                                                                   |
| source             |                      string                      |     true     | Name of the source the SQL should execute on.                                                                                              |
| description        |                      string                      |     true     | Description of the tool that is passed to the LLM.                                                                                         |
| statement          |                      string                      |     true     | SQL statement to execute on.                                                                                                               |
| parameters         |    [parameters](..#specifying-parameters)    |    false     | List of [parameters](..#specifying-parameters) that will be inserted into the SQL statement.                                           |
| templateParameters | [templateParameters](..#template-parameters) |    false     | List of [templateParameters](..#template-parameters) that will be inserted into the SQL statement before executing prepared statement. |
