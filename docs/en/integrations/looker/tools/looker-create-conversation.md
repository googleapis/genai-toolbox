---
title: "looker-create-conversation Tool"
type: docs
weight: 1
description: >
  "looker-create-conversation" tool starts a new conversation session with an AI agent.

---

## About

The `looker-create-conversation` tool initiates a new conversation session
associated with a specific Looker AI agent.

Creating a conversation is the first step in interacting with an AI agent.
The tool returns a conversation object, including a `conversation_id` which
is required for subsequent messages and chat interactions.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

- `agent_id` (required): The unique ID of the AI agent to associate with the conversation.
- `name` (optional): A descriptive name for the conversation.

## Example

```yaml
kind: tool
name: create_conversation
type: looker-create-conversation
source: looker-source
description: |
  Start a new conversation session with an AI agent.
  Required Parameter:
  - agent_id: The ID of the agent to use.
  Optional Parameter:
  - name: A name to identify this conversation.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-create-conversation"               |
| source      |  string  |     true     | Name of the Looker source to use.                  |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
