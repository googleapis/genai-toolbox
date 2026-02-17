---
title: "looker-conversational-analytics"
type: docs
weight: 1
description: >
  The "looker-conversational-analytics" tool will use the Conversational
  Analaytics API to analyze data from Looker
---

## About

A `looker-conversational-analytics` tool allows you to ask questions about your
Looker data.


`looker-conversational-analytics` accepts two parameters:

1. `user_query_with_context`: The question asked of the Conversational Analytics
   system.
2. `explore_references`: A list of one to five explores that can be queried to
   answer the question. The form of the entry is `[{"model": "model name",
   "explore": "explore name"}, ...]`

## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: ask_data_insights
type: looker-conversational-analytics
source: looker-source
description: |
  Use this tool to ask questions about your data using the Looker Conversational
  Analytics API. You must provide a natural language query and a list of
  1 to 5 model and explore combinations (e.g. [{'model': 'the_model', 'explore': 'the_explore'}]).
  Use the 'get_models' and 'get_explores' tools to discover available models and explores.
```

### Using a Data Agent

You can optionally configure a
[data agent](https://cloud.google.com/gemini/docs/conversational-analytics-api/reference/rest/v1beta/projects.locations/chat#DataAgentContext)
to provide additional context for the conversational analytics API. When a
`dataAgent` is specified, the tool uses the data agent context instead of inline
context.

```yaml
kind: tools
name: ask_data_insights
type: looker-conversational-analytics
source: looker-source
dataAgent: projects/my-project/locations/us/dataAgents/my-agent-id
description: |
  Use this tool to ask questions about your data using the Looker Conversational
  Analytics API. You must provide a natural language query and a list of
  1 to 5 model and explore combinations (e.g. [{'model': 'the_model', 'explore': 'the_explore'}]).
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "lookerca-conversational-analytics".       |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
| dataAgent   |  string  |    false     | Full resource name of a data agent (e.g. `projects/{project}/locations/{location}/dataAgents/{dataAgentId}`). When set, the tool uses data agent context instead of inline context. |
