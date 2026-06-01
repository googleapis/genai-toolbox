---
title: "looker-create-agent Tool"
type: docs
weight: 1
description: >
  "looker-create-agent" creates a Looker Conversation Analytics agent.
---

## About

The `looker-create-agent` tool allows LLMs to create a Looker Agent using the Looker Go SDK.

```json
{
  "name": "looker-create-agent",
  "parameters": {
    "name": "My Agent",
    "instructions": "You are a helpful assistant.",
    "sources": [{"model": "my_model", "explore": "my_explore"}],
    "code_interpreter": true
  }
}
```

## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: create_agent
type: looker-create-agent
source: my-looker-instance
description: |
  Create a new Looker agent.
  - `name` (string): The name of the agent.
  - `description` (string): The description of the agent.
  - `instructions` (string): The instructions (system prompt) for the agent.
  - `sources` (array): Optional. A list of JSON-encoded data sources for the agent (e.g., `[{"model": "my_model", "explore": "my_explore"}]`).
  - `code_interpreter` (boolean): Optional. Enables Code Interpreter for this Agent.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-create-agent".                     |
| source      |  string  |     true     | Name of the Looker source.                         |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
