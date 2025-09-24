---
title: "lookerca-conversational-analytics
type: docs
weight: 1
description: >
  The "lookerca-conversational-analytics" tool will use the Conversational
  Analaytics API to analyze data from Looker
aliases:
- /resources/tools/lookerca-conversational-analytics
---

## About

A `lookerca-conversational-analytics` tool allows you to ask questions about your Looker data.

It's compatible with the following sources:

- [lookerca](../../sources/lookerca.md)

`lookerca-conversational-analytics` accepts two parameters:

1. `user_query_with_context`: The question asked of the Conversational Analytics system.
2. `explore_references`: A list of one to five explores that can be queried to answer the
   question. The form of the entry is `[{"model": "model name", "explore": "explore name"}, ...]`

## Example

```yaml
tools:
    ask_data_insights:
        kind: lookerca-conversational-analytics
        source: lookerca-source
        description: |
          Use this tool to perform data analysis, get insights,
          or answer complex questions about the contents of specific
          Looker explores.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "lookerca-conversational-analytics".                                                     |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
