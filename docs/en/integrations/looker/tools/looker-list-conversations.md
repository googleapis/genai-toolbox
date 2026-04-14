---
title: "looker-list-conversations Tool"
type: docs
weight: 1
description: >
  "looker-list-conversations" tool lists existing conversation sessions.

---

## About

The `looker-list-conversations` tool retrieves a list of previously
created conversation sessions.

Users can optionally filter the results by `agent_id` or use `limit` and
`offset` for pagination.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

- `agent_id` (optional): Filter conversations by a specific AI agent ID.
- `limit` (optional): The maximum number of conversations to fetch. Default is 100.
- `offset` (optional): The number of conversations to skip before fetching. Default is 0.

## Example

```yaml
kind: tool
name: list_conversations
type: looker-list-conversations
source: looker-source
description: |
  Search for existing conversation sessions.
  Optional Parameters:
  - agent_id: Filter by agent ID.
  - limit: Pagination limit.
  - offset: Pagination offset.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-list-conversations"                |
| source      |  string  |     true     | Name of the Looker source to use.                  |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
