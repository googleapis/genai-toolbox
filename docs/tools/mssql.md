# Cloud SQL Mssql Tool

A "mssql" tool executes a pre-defined SQL statement against a Cloud SQL for SQL Server
database. It's compatible with any of the following sources:

- [cloud-sql-mssql](../sources/cloud-sql-mssql.md)

Toolbox supports the [prepare statement syntax][prepare-statement] of MS SQL
Server and expects parameters in the SQL query to be in the form of either @Name
or @p1 to @pN (ordinal position).

```sql
db.QueryContext(ctx, `select * from t where ID = @ID and Name = @p2;`, sql.Named("ID", 6), "Bob")
```

[prepare-statement]: https://learn.microsoft.com/sql/relational-databases/system-stored-procedures/sp-prepare-transact-sql?view=sql-server-ver16

## Example

```yaml
tools:
 search_flights_by_number:
    kind: mssql
    source: my-instance
    statement: |
      SELECT * FROM flights
      WHERE airline = @airline
      AND flight_number = @number
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
      - name: number
        type: string
        description: 1 to 4 digit number
```

## Reference

| **field**   |                   **type**                   | **required** | **description**                                                                                     |
|-------------|:--------------------------------------------:|:------------:|-----------------------------------------------------------------------------------------------------|
| kind        |                    string                    |     true     | Must be "mssql". |
| source      |                    string                    |     true     | Name of the source the T-SQL statement should execute on.|
| description |                    string                    |     true     | Description of the tool that is passed to the LLM|
| statement   |                    string                    |     true     | SQL statement to execute. |
| parameters  | [parameter](README.md#specifying-parameters) |     true     | List of [parameters](README.md#specifying-parameters) that will be inserted into the SQL statement. |
