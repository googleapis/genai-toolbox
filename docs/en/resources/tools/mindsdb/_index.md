---
title: "MindsDB Tools"
type: docs
weight: 1
description: >
  MindsDB tools that enable SQL queries across hundreds of datasources and ML models.
---

## About

MindsDB is the most widely adopted AI federated database that enables you to query hundreds of datasources and ML models through a single SQL interface. The following tools work with MindsDB databases:

- [mindsdb-execute-sql](mindsdb-execute-sql.md) - Execute SQL queries directly on MindsDB
- [mindsdb-sql](mindsdb-sql.md) - Execute parameterized SQL queries on MindsDB

These tools leverage MindsDB's capabilities to:
- **Connect to Multiple Datasources**: Query databases, APIs, file systems, and more through SQL
- **Cross-Datasource Operations**: Perform joins and analytics across different data sources
- **ML Model Integration**: Use trained ML models as virtual tables for predictions
- **Unstructured Data Processing**: Query documents, images, and other unstructured data as structured tables
- **Real-time Predictions**: Get real-time predictions from ML models through SQL
- **API Translation**: Automatically translate SQL queries into REST APIs, GraphQL, and native protocols

## Supported Datasources

MindsDB automatically translates your SQL queries into the appropriate APIs for hundreds of datasources:

### **Business Applications**
- **Salesforce**: Query leads, opportunities, accounts, and custom objects
- **Jira**: Access issues, projects, workflows, and team data  
- **GitHub**: Query repositories, commits, pull requests, and issues
- **Slack**: Access channels, messages, and team communications
- **HubSpot**: Query contacts, companies, deals, and marketing data

### **Databases & Storage**
- **MongoDB**: Query NoSQL collections as structured tables
- **PostgreSQL/MySQL**: Standard relational databases
- **Redis**: Key-value stores and caching layers
- **Elasticsearch**: Search and analytics data
- **S3/Google Cloud Storage**: File storage and data lakes

### **Communication & Email**
- **Gmail/Outlook**: Query emails, attachments, and metadata
- **Microsoft Teams**: Team communications and files
- **Discord**: Server data and message history

### **Analytics & Monitoring**
- **Google Analytics**: Website traffic and user behavior
- **Mixpanel**: Product analytics and user events
- **Datadog**: Infrastructure monitoring and logs
- **Grafana**: Time-series data and metrics

## Example Use Cases

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
WHERE s.stage = 'Closed Won';
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
WHERE e.date >= '2024-01-01';
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

Since MindsDB implements the MySQL wire protocol, these tools are functionally compatible with MySQL tools while providing access to MindsDB's advanced federated database capabilities.

## Working Configuration Example

Here's a complete working configuration that has been tested:

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