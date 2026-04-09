---
title: "looker-list-agents Tool"
type: docs
weight: 1
description: >
  "looker-list-agents" retrieves the list of Looker Conversation Analytics agents.
---

## About

The `looker-list-agents` tool allows LLMs to list Looker Agents using the Looker Go SDK.

```json
{
  "name": "looker-list-agents"
}
```

## Compatible Sources

{{< compatible-sources >}}

## Example

To use the `looker-list-agents` tool, you must define it in your `server.yaml` file.

```yaml
kind: tool
name: list_agents
type: looker-list-agents
source: my-looker-instance
description: |
  List all Looker agents.
  This tool takes no parameters.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "looker-list-agents".                      |
| source      |  string  |     true     | Name of the Looker source.                         |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
