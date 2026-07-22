---
title: "conversational-analytics-create-data-agent"
type: docs
weight: 1
description: >
  A "conversational-analytics-create-data-agent" tool allows creating a new Conversational Analytics data agent.
aliases:
- /resources/tools/conversational-analytics-create-data-agent
---

## About

A `conversational-analytics-create-data-agent` tool allows you to create
a new data agent in Conversational Analytics.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

`conversational-analytics-create-data-agent` accepts the following parameters:

- **`data_agent_id`:** The ID to use for the new data agent.
- **`payload`:** The JSON string representation of the DataAgent resource to create.

## Example

```yaml
tools:
  create_agent:
    kind: conversational-analytics-create-data-agent
    source: my-conversational-analytics-source
    location: global
    description: |
      Use this tool to create a new data agent with the specified configuration payload.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "conversational-analytics-create-data-agent". |
| source      |  string  |     true     | Name of the source.                                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
| location    |  string  |    false     | The Google Cloud location (default: "global").     |
