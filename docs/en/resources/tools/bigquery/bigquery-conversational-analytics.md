---
title: "bigquery-conversational-analytics"
type: docs
weight: 1
description: > 
  A "bigquery-conversational-analytics" tool allows conversational interaction with a BigQuery source.
aliases:
- /resources/tools/bigquery-conversational-analytics
---

## About

A `bigquery-conversational-analytics` tool allows you to ask questions about your data in natural language. 

This function takes a user's question (which can include conversational history for context) 
and references to specific BigQuery tables, and sends them to a stateless conversational API.

The API uses a GenAI agent to understand the question, generate and execute SQL queries 
and Python code, and formulate an answer. This function returns a detailed, sequential 
log of this entire process, which includes any generated SQL or Python code, the data 
retrieved, and the final text answer.

When the `allowed_datasets` parameter is configured in the `bigquery` source, this tool 
will enforce that all tables provided in the `table_references` parameter belong to one 
of the datasets in the allowed list. If any table is from a dataset that is not in the
list, the tool will return an error.

**Note**: This tool requires additional setup in your project. Please refer to the 
official [Conversational Analytics API documentation](https://cloud.google.com/gemini/docs/conversational-analytics-api/overview)
for instructions.

It's compatible with the following sources:

- [bigquery](../sources/bigquery.md)

The tool takes the following input parameters:

*   `user_query_with_context`: The user's question, potentially including conversation history and system instructions for context.
*   `table_references`: A JSON string of a list of BigQuery tables to use as context. Each object in the list must contain `projectId`, `datasetId`, and `tableId`. Example: `'[{"projectId": "my-gcp-project", "datasetId": "my_dataset", "tableId": "my_table"}]'`

## Example

```yaml
tools:
  ask_data_insights:
    kind: bigquery-conversational-analytics
    source: my-bigquery-source
    description: |
      Use this tool to perform data analysis, get insights, or answer complex 
      questions about the contents of specific BigQuery tables.
```

## Reference
| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "bigquery-conversational-analytics".                                                     |
| source      |                   string                   |     true     | Name of the source for chat.                                                    |
| description |                   string                   |     true     | Description of the tool 
that is passed to the LLM.                                               |
