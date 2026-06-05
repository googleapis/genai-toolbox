---
title: "looker-get-agent Tool"
type: docs
weight: 1
description: >
  "looker-get-agent" retrieves a Looker Conversation Analytics agent.
---

## About

The `looker-get-agent` tool allows LLMs to retrieve a specific Looker Agent by ID using the Looker Go SDK.

To use the `looker-get-agent` tool, you must define it in your `server.yaml` file.

```json
{
  "name": "looker-get-agent",
  "parameters": {
    "agent_id": "123"
  }
}
```

## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: get_agent
type: looker-get-agent
source: my-looker-instance
description: |
  Retrieve a Looker agent.
  - `agent_id` (string): The ID of the agent.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-get-agent".                        |
| source      |  string  |     true     | Name of the Looker source.                         |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
