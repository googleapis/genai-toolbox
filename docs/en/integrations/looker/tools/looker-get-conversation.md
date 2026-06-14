---
title: "looker-get-conversation"
type: docs
weight: 1
description: >
  "looker-get-conversation" tool retrieves detailed information about a specific conversation.

---

## About

The `looker-get-conversation` tool retrieves details for a single conversation
by its ID.

This is useful for checking the current status or metadata of a
conversation before interacting with it.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

- `conversation_id` (required): The unique ID of the conversation to retrieve.

## Example

```yaml
kind: tool
name: get_conversation
type: looker-get-conversation
source: looker-source
description: |
  Retrieve detailed information about a single conversation.
  Required Parameter:
  - conversation_id: The ID of the conversation to get.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-get-conversation"                  |
| source      |  string  |     true     | Name of the Looker source to use.                  |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
