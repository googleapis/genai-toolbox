---
title: "Gemini Data Analytics QueryData"
type: docs
weight: 1
description: >
  A tool to convert natural language queries into SQL statements using the Gemini Data Analytics QueryData API.
aliases:
  - /resources/tools/cloud-gemini-data-analytics-query
---

## About

The `cloud-gemini-data-analytics-query` tool allows you to send natural language questions to the Gemini Data Analytics API and receive structured responses containing SQL queries, natural language answers, and explanations. See https://docs.cloud.google.com/gemini/docs/conversational-analytics-api/data-agent-authored-context-databases for details.

## Example

```yaml
tools:
  my-gda-query-tool:
    kind: cloud-gemini-data-analytics-query
    source: my-gda-source
    description: "Use this tool to send natural language queries to the Gemini Data Analytics API and receive SQL, natural language answers, and explanations."
    location: ${your_database_location}
    context:
      datasourceReferences:
        cloudSqlReference:
          databaseReference:
            projectId: "${your_project_id}"
            region: "${your_database_instance_region}"
            instanceId: "${your_database_instance_id}"
            databaseId: "${your_database_name}"
            engine: "POSTGRESQL"
          agentContextReference:
            contextSetId: "${your_context_set_id}" # E.g. projects/${project_id}/locations/${context_set_location}/contextSets/${context_set_id}
    generationOptions:
      generateQueryResult: true
      generateNaturalLanguageAnswer: true
      generateExplanation: true
      generateDisambiguationQuestion: true
```

### Usage Flow

When using this tool, a `prompt` parameter containing a natural language query is provided to the tool (typically by an agent). The tool then interacts with the Gemini Data Analytics API using the context defined in your configuration.

The structure of the response depends on the `generationOptions` configured in your tool definition (e.g., enabling `generateQueryResult` will include the SQL query results).

See https://docs.cloud.google.com/gemini/docs/conversational-analytics-api/reference/rest for details.

**Example Input Prompt:**

```text
How many accounts who have region in Prague are eligible for loans? A3 contains the data of region.
```

**Example API Response:**

```json
{
  "generatedQuery": "SELECT COUNT(T1.account_id) FROM account AS T1 INNER JOIN loan AS T2 ON T1.account_id = T2.account_id INNER JOIN district AS T3 ON T1.district_id = T3.district_id WHERE T3.A3 = 'Prague'",
  "intentExplanation": "I found a template that matches the user's question. The template asks about the number of accounts who have region in a given city and are eligible for loans. The question asks about the number of accounts who have region in Prague and are eligible for loans. The template's parameterized SQL is 'SELECT COUNT(T1.account_id) FROM account AS T1 INNER JOIN loan AS T2 ON T1.account_id = T2.account_id INNER JOIN district AS T3 ON T1.district_id = T3.district_id WHERE T3.A3 = ?'. I will replace the named parameter '?' with 'Prague'.",
  "naturalLanguageAnswer": "There are 84 accounts from the Prague region that are eligible for loans.",
  "queryResult": {
    "columns": [
      {
        "type": "INT64"
      }
    ],
    "rows": [
      {
        "values": [
          {
            "value": "84"
          }
        ]
      }
    ],
    "totalRowCount": "1"
  }
}
```

## Reference

| **field**         | **type** | **required** | **description**                                                                                                                                                                                                                                              |
| ----------------- | :------: | :----------: | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| kind              |  string  |     true     | Must be "cloud-gemini-data-analytics-query".                                                                                                                                                                                                                 |
| source            |  string  |     true     | The name of the `cloud-gemini-data-analytics` source to use.                                                                                                                                                                                                 |
| description       |  string  |     true     | A description of the tool's purpose.                                                                                                                                                                                                                         |
| location          |  string  |     true     | The Google Cloud location of the target database resource (e.g., "us-central1"). This is used to construct the parent resource name in the API call.                                                                                                         |
| context           |  object  |     true     | The context for the query, including datasource references. See [QueryDataContext](https://github.com/googleapis/googleapis/blob/b32495a713a68dd0dff90cf0b24021debfca048a/google/cloud/geminidataanalytics/v1beta/data_chat_service.proto#L156) for details. |
| generationOptions |  object  |    false     | Options for generating the response. See [GenerationOptions](https://github.com/googleapis/googleapis/blob/b32495a713a68dd0dff90cf0b24021debfca048a/google/cloud/geminidataanalytics/v1beta/data_chat_service.proto#L135) for details.                       |
