---
title: "looker-conversational-analytics"
type: docs
weight: 1
description: >
  The "looker-conversational-analytics" tool will use the Conversational
  Analaytics API to analyze data from Looker
aliases:
- /resources/tools/looker-conversational-analytics
---

## About

A `looker-conversational-analytics` tool allows you to ask questions about your
Looker data.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-conversational-analytics` accepts two parameters:

1. `user_query_with_context`: The question asked of the Conversational Analytics
   system.
2. `explore_references`: A list of one to five explores that can be queried to
   answer the question. The form of the entry is `[{"model": "model name",
   "explore": "explore name"}, ...]`

## Example

```yaml
tools:
    ask_data_insights:
        kind: looker-conversational-analytics
        source: looker-source
        description: |
          Use this tool to ask questions about your data using the Looker Conversational
          Analytics API. You must provide a natural language query and a list of
          1 to 5 model and explore combinations (e.g. [{'model': 'the_model', 'explore': 'the_explore'}]).
          Use the 'get_models' and 'get_explores' tools to discover available models and explores.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "lookerca-conversational-analytics".       |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
