---
title: "bigquery-chat"
type: docs
weight: 1
description: > 
  A "bigquery-chat" tool allows conversational interaction with a BigQuery source.
aliases:
- /resources/tools/bigquery-chat
---

## About

A `bigquery-chat` tool allows you to ask questions about your data in natural language. 

This function acts as a client for a "chat-with-your-data" service. It takes a user's 
question (which can include conversational history for context) and references to 
specific BigQuery tables, and sends them to a stateless conversational API.

The API uses a GenAI agent to understand the question, generate and execute SQL queries 
and Python code, and formulate an answer. This function returns a detailed, sequential 
log of this entire process, which includes any generated SQL or Python code, the data 
retrieved, and the final text answer.

**Note**: This tool requires additional setup in your project. Please refer to the 
official [Conversational Analytics API documentation](https://cloud.google.com/gemini/docs/conversational-analytics-api/overview)
for instructions.

It's compatible with the following sources:

- [bigquery](../sources/bigquery.md)


The tool takes one input parameter `question`, which is the natural language question 
you want to ask about your data.

## Example

```yaml
tools:
  chat_with_bq:
    kind: bigquery-chat
    source: my-bigquery-source
    description: |
      Use this tool to perform data analysis, get insights, or answer complex 
      questions about the contents of specific BigQuery tables.
```

## Reference
| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "bigquery-chat".                                                                  |
| source      |                   string                   |     true     | Name of the source for chat.                                                    |
| description |                   string                   |     true     | Description of the tool 
that is passed to the LLM.                                               |
