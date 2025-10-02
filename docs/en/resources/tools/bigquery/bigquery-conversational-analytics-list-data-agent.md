---
title: "bigquery-conversational-analytics-list-data-agents"
type: docs
weight: 1
description: >
  A "bigquery-conversational-analytics-list-data-agents" tool lists Conversational Analytics data agents.
aliases:
- /resources/tools/bigquery-conversational-analytics-list-data-agent
---

## About

A `bigquery-conversational-analytics-list-data-agents` tool lists all 
available Conversational Analytics data agents for a given project.

It's compatible with the following sources:

- [bigquery](../../sources/bigquery.md)

`bigquery-conversational-analytics-list-data-agents` accepts the following parameters:

- **`project`** (optional): The Google Cloud project to list data agents from. 
If not provided, it defaults to the project specified in the source configuration.

## Example

```yaml
tools:
  list_data_agents:
    kind: bigquery-conversational-analytics-list-data-agents
    source: my-bigquery-source
    description: Use this tool to list Conversational Analytics data agents.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "bigquery-conversational-analytics-list-data-agents". |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |