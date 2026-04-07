---
title: "conversational-analytics-list-accessible-data-agents"
type: docs
weight: 1
description: >
  A "conversational-analytics-list-accessible-data-agents" tool allows listing accessible Conversational Analytics data agents.
aliases:
- /resources/tools/conversational-analytics-list-accessible-data-agents
---

## About

A `conversational-analytics-list-accessible-data-agents` tool allows you to list
data agents that are accessible.

It's compatible with the following sources:

- cloud-gemini-data-analytics

`conversational-analytics-list-accessible-data-agents` does not accept any parameters.

## Example

```yaml
tools:
  list_agents:
    kind: conversational-analytics-list-accessible-data-agents
    source: my-conversational-analytics-source
    location: global
    description: |
      Use this tool to list available data agents.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "conversational-analytics-list-accessible-data-agents". |
| source      |  string  |     true     | Name of the source.                                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
| location    |  string  |    false     | The Google Cloud location (default: "global").     |