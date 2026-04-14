---
title: "looker-update-conversation Tool"
type: docs
weight: 1
description: >
  "looker-update-conversation" tool updates an existing conversation's attributes.

---

## About

The `looker-update-conversation` tool modifies attributes of an
existing conversation.

Common uses include renaming a conversation to better reflect its content.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

- `conversation_id` (required): The unique ID of the conversation to update.
- `name` (required): The new name for the conversation.

## Example

```yaml
kind: tool
name: update_conversation
type: looker-update-conversation
source: looker-source
description: |
  Update an existing conversation's name.
  Required Parameters:
  - conversation_id: The ID of the conversation.
  - name: The new name.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-update-conversation"               |
| source      |  string  |     true     | Name of the Looker source to use.                  |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
