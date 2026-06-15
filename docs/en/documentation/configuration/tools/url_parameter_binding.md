---
title: "URL Parameter Binding"
type: docs
weight: 10
description: >
  How to bind tool arguments at the transport level using URL query parameters.
---

## About

URL Parameter Binding is a transport-level feature for the SSE (HTTP) transport that allows you to bind specific arguments to tools at the connection level. This is useful for creating scoped connections for generic MCP clients where you want to restrict the client to a specific database, project, or instance without hardcoding these values in the server configuration for all users, or requiring the client to provide them.

## How It Works

When an MCP client connects to the server via SSE with query parameters (e.g., `http://localhost:5000/mcp/sse?project=my-project`):

1. **Schema Filtering**: The server automatically removes the bound parameters (like `project`) from the `inputSchema` of all tools returned by the `tools/list` endpoint. The client will not see these parameters and will not be prompted to provide them.
2. **Argument Injection**: When the client calls any tool via `tools/call`, the server automatically injects the bound values from the URL into the tool arguments before execution.
3. **Type Conversion**: Since URL query parameters are always extracted as strings, the server automatically attempts to convert the string value to the correct type if the tool parameter is defined as an `integer`, `boolean`, or `number`. Complex types like arrays or objects are not supported for automatic conversion.

This effectively abstracts the bound parameters from the client, presenting a dynamically restricted schema while enforcing execution context at the transport layer.

## Example

Assume you have a tool that requires a `project` parameter.

### 1. Connect with Scoping

The client connects to the SSE endpoint with the parameter in the URL:

```bash
curl -N "http://localhost:5000/mcp/sse?project=my-project"
```

The server returns the message endpoint with the session ID and the preserved parameter:

```text
event: endpoint
data: http://localhost:5000/mcp?sessionId=xyz&project=my-project
```

### 2. List Tools

When the client lists tools, the `project` parameter is hidden:

```json
{
  "result": {
    "tools": [
      {
        "name": "my_tool",
        "inputSchema": {
          "type": "object",
          "properties": {}, // 'project' is filtered out
          "required": []
        }
      }
    ]
  }
}
```

### 3. Call Tool

The client calls the tool without providing `project` in the body:

```json
{
  "method": "tools/call",
  "params": {
    "name": "my_tool"
    // no 'project' argument provided here
  }
}
```

The server automatically injects `project: "my-project"` and executes the tool.

## Safety Warning

> [!WARNING]
> **Never use URL parameter binding to pass sensitive credentials** like passwords, API keys, or auth tokens. URL query parameters are often logged by proxies, load balancers, and browser history, exposing them to security risks. Use this feature only for non-sensitive routing metadata like project IDs, database names, or region names.

For sensitive credentials, use the standard `Authorization` header or environment variable substitution in the `tools.yaml` configuration.
