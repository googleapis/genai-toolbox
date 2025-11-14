---
title: "mindsdb-execute-sql"
type: docs
weight: 1
description: > 
  A "mindsdb-execute-sql" tool executes a SQL statement against a MindsDB
  federated database.
aliases:
- /resources/tools/mindsdb-execute-sql
---

## About

A `mindsdb-execute-sql` tool executes a SQL statement against a MindsDB
federated database. It's compatible with any of the following sources:

- [mindsdb](../sources/mindsdb.md)

`mindsdb-execute-sql` takes one input parameter `sql` and runs the SQL
statement against the `source`. This tool enables you to:

- **Query Multiple Datasources**: Execute SQL across hundreds of connected
  datasources
- **Cross-Datasource Joins**: Perform joins between different databases, APIs,
  and file systems
- **ML Model Predictions**: Query ML models as virtual tables for real-time
  predictions
- **Unstructured Data**: Query documents, images, and other unstructured data as
  structured tables
- **Federated Analytics**: Perform analytics across multiple datasources
  simultaneously
- **API Translation**: Automatically translate SQL queries into REST APIs,
  GraphQL, and native protocols

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
WHERE s.stage = 'Closed Won'
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
WHERE e.date >= '2024-01-01'
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
WHERE predicted_churn_probability > 0.8;
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
WHERE department = 'Engineering'
ORDER BY created_at DESC;
```

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
 execute_sql_tool:
    kind: mindsdb-execute-sql
    source: my-mindsdb-instance
    description: Use this tool to execute SQL statements across multiple datasources and ML models.
```

### Working Configuration Example

Here's a working configuration that has been tested:

```yaml
sources:
  my-pg-source:
    kind: mindsdb
    host: 127.0.0.1
    port: 47335
    database: files
    user: mindsdb

tools:
  mindsdb-execute-sql:
    kind: mindsdb-execute-sql
    source: my-pg-source
    description: |
      Execute SQL queries directly on MindsDB database.
      Use this tool to run any SQL statement against your MindsDB instance.
      Example: SELECT * FROM my_table LIMIT 10
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "mindsdb-execute-sql".                     |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
