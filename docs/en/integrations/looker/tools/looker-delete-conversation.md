---
title: "looker-delete-conversation Tool"
type: docs
weight: 1
description: >
  "looker-delete-conversation" tool deletes a conversation session.

---

## About

The `looker-delete-conversation` tool removes a conversation session.

Once deleted, the conversation and its message history are no longer
accessible via the API.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

- `conversation_id` (required): The unique ID of the conversation to delete.

## Example

```yaml
kind: tool
name: delete_conversation
type: looker-delete-conversation
source: looker-source
description: |
  Delete a conversation and its history.
  Required Parameter:
  - conversation_id: The ID of the conversation.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-delete-conversation"               |
| source      |  string  |     true     | Name of the Looker source to use.                  |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
