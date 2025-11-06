---
title: "MindsDB"
type: docs
weight: 1
description: >
  MindsDB is an AI federated database that enables SQL queries across hundreds of datasources and ML models.
---

## About

[MindsDB][mindsdb-docs] is an AI federated database in the world. It allows you to combine information from hundreds of datasources as if they were SQL, supporting joins across datasources and enabling you to query all unstructured data as if it were structured.

MindsDB translates MySQL queries into whatever API is needed - whether it's REST APIs, GraphQL, or native database protocols. This means you can write standard SQL queries and MindsDB automatically handles the translation to APIs like Salesforce, Jira, GitHub, email systems, MongoDB, and hundreds of other datasources.

MindsDB also enables you to use ML frameworks to train and use models as virtual tables from the data in those datasources. With MindsDB, the GenAI Toolbox can now expand to hundreds of datasources and leverage all of MindsDB's capabilities on ML and unstructured data.

**Key Features:**
- **Federated Database**: Connect and query hundreds of datasources through a single SQL interface
- **Cross-Datasource Joins**: Perform joins across different datasources seamlessly
- **API Translation**: Automatically translates MySQL queries into REST APIs, GraphQL, and native protocols
- **Unstructured Data Support**: Query unstructured data as if it were structured
- **ML as Virtual Tables**: Train and use ML models as virtual tables
- **MySQL Wire Protocol**: Compatible with standard MySQL clients and tools

[mindsdb-docs]: https://docs.mindsdb.com/
[mindsdb-github]: https://github.com/mindsdb/mindsdb

## Supported Datasources

MindsDB supports hundreds of datasources, including:

### **Business Applications**
- **Salesforce**: Query leads, opportunities, accounts, and custom objects
- **Jira**: Access issues, projects, workflows, and team data
- **GitHub**: Query repositories, commits, pull requests, and issues
- **Slack**: Access channels, messages, and team communications
- **HubSpot**: Query contacts, companies, deals, and marketing data

### **Databases & Storage**
- **MongoDB**: Query NoSQL collections as structured tables
- **Redis**: Key-value stores and caching layers
- **Elasticsearch**: Search and analytics data
- **S3/Google Cloud Storage**: File storage and data lakes

### **Communication & Email**
- **Gmail/Outlook**: Query emails, attachments, and metadata
- **Slack**: Access workspace data and conversations
- **Microsoft Teams**: Team communications and files
- **Discord**: Server data and message history



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

## Requirements

### Database User

This source uses standard MySQL authentication since MindsDB implements the MySQL wire protocol. You will need to [create a MindsDB user][mindsdb-users] to login to the database with. If MindsDB is configured without authentication, you can omit the password field.

[mindsdb-users]: https://docs.mindsdb.com/
## Example

```yaml
sources:
    my-mindsdb-source:
        kind: mindsdb
        host: 127.0.0.1
        port: 3306
        database: my_db
        user: ${USER_NAME}
        password: ${PASSWORD} # Optional: omit if MindsDB is configured without authentication
        queryTimeout: 30s # Optional: query timeout duration
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
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Use Cases

With MindsDB integration, you can:

- **Query Multiple Datasources**: Connect to databases, APIs, file systems, and more through a single SQL interface
- **Cross-Datasource Analytics**: Perform joins and analytics across different data sources
- **ML Model Integration**: Use trained ML models as virtual tables for predictions and insights
- **Unstructured Data Processing**: Query documents, images, and other unstructured data as structured tables
- **Real-time Predictions**: Get real-time predictions from ML models through SQL queries
- **API Abstraction**: Write SQL queries that automatically translate to REST APIs, GraphQL, and native protocols

## Reference

| **field**    | **type** | **required** | **description**                                                                                 |
| ------------ | :------: | :----------: | ----------------------------------------------------------------------------------------------- |
| kind         |  string  |     true     | Must be "mindsdb".                                                                              |
| host         |  string  |     true     | IP address to connect to (e.g. "127.0.0.1").                                                    |
| port         |  string  |     true     | Port to connect to (e.g. "3306").                                                               |
| database     |  string  |     true     | Name of the MindsDB database to connect to (e.g. "my_db").                                      |
| user         |  string  |     true     | Name of the MindsDB user to connect as (e.g. "my-mindsdb-user").                                |
| password     |  string  |    false     | Password of the MindsDB user (e.g. "my-password"). Optional if MindsDB is configured without authentication. |
| queryTimeout |  string  |    false     | Maximum time to wait for query execution (e.g. "30s", "2m"). By default, no timeout is applied. |

## Resources

- [MindsDB Documentation][mindsdb-docs] - Official documentation and guides
- [MindsDB GitHub][mindsdb-github] - Source code and community 