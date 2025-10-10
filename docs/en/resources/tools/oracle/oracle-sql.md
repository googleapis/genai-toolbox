---
title: "oracle-sql"
type: docs
weight: 1
description: > 
  An "oracle-sql" tool executes a pre-defined SQL statement against an Oracle database.
aliases:
- /resources/tools/oracle-sql
---

## About

An `oracle-sql` tool executes a pre-defined SQL statement against an
Oracle database. It's compatible with the following source:

- [oracle](../../sources/oracle.md)

The specified SQL statement is executed using [prepared statements][oracle-stmt]
for security and performance. It expects parameter placeholders in the SQL query
to be in the native Oracle format (e.g., `:1`, `:2`).

[oracle-stmt]: https://docs.oracle.com/javase/tutorial/jdbc/basics/prepared.html

## Example

> **Note:** This tool uses parameterized queries to prevent SQL injections.
> Query parameters can be used as substitutes for arbitrary expressions.
> Parameters cannot be used as substitutes for identifiers, column names, table
> names, or other parts of the query.

```yaml
tools:
  search_flights_by_number:
    kind: oracle-sql
    source: my-oracle-instance
    statement: |
      SELECT * FROM flights
      WHERE airline = :1
      AND flight_number = :2
      FETCH FIRST 10 ROWS ONLY
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