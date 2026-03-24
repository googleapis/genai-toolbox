---
title: "looker-agent"
type: docs
weight: 1
description: >
  Manage Looker Agents
aliases:
- /resources/tools/looker-agent
---

# Looker Agent

The `looker-agent` tool allows LLMs to manage Looker Agents. It supports listing, retrieving, creating, and deleting agents using the Looker Go SDK.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

## Configuration

To use the `looker-agent` tool, you must define it in your `server.yaml` file.

```yaml
kind: tool
name: agent_manage
type: looker-agent
source: my_looker_source
description: |
  Manage Looker AI Agents. This tool allows you to perform various operations on Looker Agents, 
  including listing all available agents, retrieving details for a specific agent, 
  creating a new agent, and deleting an existing one.

  Parameters:
  - operation (required): The action to perform.
    - 'list': Returns a list of all existing agents. No other parameters needed.
    - 'get': Retrieves detailed information about a specific agent. Requires 'agent_id'.
    - 'create': Creates a new Looker AI Agent. Requires 'name'.
    - 'delete': Removes an existing agent. Requires 'agent_id'.
  - agent_id (optional): The unique identifier of the agent. Required for 'get' and 'delete' operations.
  - name (optional): The display name for the new agent. Required for 'create' operation.
```

## Parameters

| **Parameter** | **Type** | **Required** | **Description** |
|:-------------|:--------:|:------------:|:----------------|
| `operation` | `string` | Yes | The operation to perform. Must be one of: `list`, `get`, `create`, or `delete`. |
| `agent_id` | `string` | No | The ID of the agent. Required for `get` and `delete` operations. |
| `name` | `string` | No | The name of the agent. Required for `create` operation. |

## Operations

### List Agents
Retrieve a list of all agents.
```json
{
  "operation": "list"
}
```

### Get Agent
Retrieve details of a specific agent by its ID.
```json
{
  "operation": "get",
  "agent_id": "12345"
}
```

### Create Agent
Create a new agent with the given name.
```json
{
  "operation": "create",
  "name": "My AI Assistant"
}
```

### Delete Agent
Delete an agent by its ID.
```json
{
  "operation": "delete",
  "agent_id": "12345"
}
```

## Dependencies
This tool requires the underlying Looker Go SDK to support the `Agent` API resource (v0.26.6+).
