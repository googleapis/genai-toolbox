---
title: "URL Parameter Binding"
type: docs
weight: 10
description: >
  How to bind tool arguments at the transport level using URL query parameters.
---

## About

URL Parameter Binding is a transport-level feature for HTTP-based transports (such as SSE or standard HTTP POST endpoints) that allows you to bind specific arguments to tools at the connection or request level. This is useful for creating scoped connections for generic MCP clients where you want to restrict the client to a specific database, project, or instance without hardcoding these values in the server configuration for all users, or requiring the client to provide them.

## How It Works

When an MCP client connects or sends requests to the server via HTTP with query parameters (e.g., `http://localhost:5000/mcp/sse?project=my-project` or `http://localhost:5000/mcp?project=my-project`):

1. **Schema Filtering**: The server automatically removes the bound parameters (like `project`) from the `inputSchema` of all tools returned by the `tools/list` endpoint. The client will not see these parameters and will not be prompted to provide them.
2. **Argument Injection**: When the client calls any tool via `tools/call`, the server automatically injects the bound values from the URL into the tool arguments before execution.
3. **Type Conversion**: Since URL query parameters are always extracted as strings, the server automatically attempts to convert the string value to the correct type based on the tool's parameter definition. It supports:
   - **Simple types**: `integer`, `boolean`, and `number` (parsed from their string representation).
   - **Complex types**: `array` and `object` (parsed from JSON-encoded string values, e.g. `?my_array=%5B%22a%22%2C%22b%22%5D` or `?my_map=%7B%22k%22%3A%22v%22%7D`).

This effectively abstracts the bound parameters from the client, presenting a dynamically restricted schema while enforcing execution context at the transport layer.

## Example

Assume you have a tool that requires a `project` parameter.

### 1. Connect or Request with Scoping

The client connects to the SSE endpoint or sends a request to the standard HTTP POST endpoint with the parameter in the URL.

For SSE, the client establishes a connection:

```bash
curl -N "http://localhost:5000/mcp/sse?project=my-project"
```

The server returns the message endpoint with the session ID and the preserved parameter:

```text
event: endpoint
data: http://localhost:5000/mcp?sessionId=xyz&project=my-project
```

For standard HTTP POST requests, the client directly includes query parameters in the request URL:

```bash
curl -X POST "http://localhost:5000/mcp?project=my-project" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}'
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

## Using with MCP Toolbox Client SDKs

When using the MCP Toolbox Client SDKs, you can pass URL parameters directly in the server connection string. The SDK will automatically retain these parameters when communicating with the server.

### Python

```python
from toolbox_core import ToolboxClient

# Initialize the client with the full scoped connection string
# The `project` parameter will be sent with all MCP requests
client = ToolboxClient("http://localhost:5000?project=my-project")

# When loading tools, the 'project' parameter is automatically omitted 
# from the required schema by the server
tool = await client.load_tool("my_tool")

# The server will automatically inject `project: "my-project"` upon execution
result = await tool(other_param="value")
```

## Safety Warning

> [!WARNING]
> **Never use URL parameter binding to pass sensitive credentials** like passwords, API keys, or auth tokens. URL query parameters are often logged by proxies, load balancers, and browser history, exposing them to security risks. Use this feature only for non-sensitive routing metadata like project IDs, database names, or region names.

For sensitive credentials, use the standard `Authorization` header or environment variable substitution in the `tools.yaml` configuration.
