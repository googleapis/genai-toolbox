---
title: "mindsdb-sql"
type: docs
weight: 1
description: > 
  A "mindsdb-sql" tool executes a pre-defined SQL statement against a MindsDB
  federated database.
aliases:
- /resources/tools/mindsdb-sql
---

## About

A `mindsdb-sql` tool executes a pre-defined SQL statement against a MindsDB
federated database. It's compatible with any of the following sources:

- [mindsdb](../sources/mindsdb.md)

The specified SQL statement is executed as a [prepared statement][mysql-prepare],
and expects parameters in the SQL query to be in the form of placeholders `?`.

This tool enables you to:
- **Query Multiple Datasources**: Execute parameterized SQL across hundreds of connected datasources
- **Cross-Datasource Joins**: Perform joins between different databases, APIs, and file systems
- **ML Model Predictions**: Query ML models as virtual tables for real-time predictions
- **Unstructured Data**: Query documents, images, and other unstructured data as structured tables
- **Federated Analytics**: Perform analytics across multiple datasources simultaneously
- **API Translation**: Automatically translate SQL queries into REST APIs, GraphQL, and native protocols

[mysql-prepare]: https://dev.mysql.com/doc/refman/8.4/en/sql-prepared-statements.html

## Example Queries

### Cross-Datasource Analytics
```sql
-- Join Salesforce opportunities with GitHub activity
SELECT 
    s.opportunity_name,
    s.amount,
    g.repository_name,
    COUNT(g.commits) as commit_count
FROM salesforce.opportunities s
JOIN github.repositories g ON s.account_id = g.owner_id
WHERE s.stage = ?
GROUP BY s.opportunity_name, s.amount, g.repository_name;
```

### Email & Communication Analysis
```sql
-- Analyze email patterns with Slack activity
SELECT 
    e.sender,
    e.subject,
    s.channel_name,
    COUNT(s.messages) as message_count
FROM gmail.emails e
JOIN slack.messages s ON e.sender = s.user_name
WHERE e.date >= ?
GROUP BY e.sender, e.subject, s.channel_name;
```

### ML Model Predictions
```sql
-- Use ML model to predict customer churn
SELECT 
    customer_id,
    customer_name,
    predicted_churn_probability,
    recommended_action
FROM customer_churn_model
WHERE predicted_churn_probability > ?;
```

### MongoDB Query
```sql
-- Query MongoDB collections as structured tables
SELECT 
    name,
    email,
    department,
    created_at
FROM mongodb.users
WHERE department = ?
ORDER BY created_at DESC;
```

## Example

> **Note:** This tool uses parameterized queries to prevent SQL injections.
> Query parameters can be used as substitutes for arbitrary expressions.
> Parameters cannot be used as substitutes for identifiers, column names, table
> names, or other parts of the query.

```yaml
tools:
 search_flights_by_number:
    kind: mindsdb-sql
    source: my-mindsdb-instance
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
> [templateParameters](_index#template-parameters).

```yaml
tools:
 list_table:
    kind: mindsdb-sql
    source: my-mindsdb-instance
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
| kind               |                   string                         |     true     | Must be "mindsdb-sql".                                                                                                                     |
| source             |                   string                         |     true     | Name of the source the SQL should execute on.                                                                                              |
| description        |                   string                         |     true     | Description of the tool that is passed to the LLM.                                                                                         |
| statement          |                   string                         |     true     | SQL statement to execute on.                                                                                                               |
| parameters         | [parameters](_index#specifying-parameters)       |    false     | List of [parameters](_index#specifying-parameters) that will be inserted into the SQL statement.                                           |
| templateParameters | [templateParameters](_index#template-parameters) |    false     | List of [templateParameters](_index#template-parameters) that will be inserted into the SQL statement before executing prepared statement. | 