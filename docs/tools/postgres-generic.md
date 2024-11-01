# PostgreSQL Generic Tool 

The "postgres-generic" tool executes a pre-defined SQL statement. 

## Requirements 

PostgreSQL Generic Tools require one of the following sources:
- [postgres](../sources/postgres.md)

## Example

```yaml
tools:
 search_flights_by_number:
    kind: postgres-generic
    source: my-pg-instance
    statement: |
      SELECT * FROM flights
      WHERE airline = $1
      AND flight_number = $2
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

## Reference

| **field**   |                   **type**                   | **required** | **description**                                                                                     |
|-------------|:--------------------------------------------:|:------------:|-----------------------------------------------------------------------------------------------------|
| kind        |                    string                    |     true     | Must be "postgres-generic".                                                                         |
| source      |                    string                    |     true     | Name of the source the SQL should execute on.                                                       |
| description |                    string                    |     true     | Port to connect to (e.g. "5432")                                                                    |
| statement   |                    string                    |     true     | SQL statement to execute on.                                                                        |
| parameters  | [parameter](README.md#specifying-parameters) |     true     | List of [parameters](README.md#specifying-parameters) that will be inserted into the SQL statement. |


