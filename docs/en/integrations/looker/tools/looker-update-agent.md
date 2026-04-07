---
title: "looker-update-agent Tool"
type: docs
weight: 1
description: >
  "looker-update-agent" updates a Looker Conversation Analytics agent.
---

## About

The `looker-update-agent` tool allows LLMs to update an existing Looker Agent using the Looker Go SDK.

```json
{
  "name": "looker-update-agent",
  "parameters": {
    "agent_id": "123",
    "name": "Updated Agent Name"
  }
}
```

## Compatible Sources

{{< compatible-sources >}}

## Example

To use the `looker-update-agent` tool, you must define it in your `server.yaml` file.

```yaml
kind: tool
name: update_agent
type: looker-update-agent
source: my-looker-instance
description: |
  Update a Looker agent.
  - `agent_id` (string): The ID of the agent.
  - `name` (string): The name of the agent.
  - `description` (string): The description of the agent.
  - `instructions` (string): The instructions (system prompt) for the agent.
  - `sources` (array): Optional. A list of JSON-encoded data sources for the agent (e.g., `[{"model": "my_model", "explore": "my_explore"}]`).
  - `code_interpreter` (boolean): Optional. Enables Code Interpreter for this Agent.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-update-agent".                     |
| source      |  string  |     true     | Name of the Looker source.                         |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
