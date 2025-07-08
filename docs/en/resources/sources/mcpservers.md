---
title: "Model Context Protocol Servers"
linkTitle: "MCP Servers"
type: docs
weight: 1
description: >
  Supporting Anthropic MCP-compatible external servers.
---

## About

[Model Context Protocol - MCP][mcp-spec] follows a client-server architecture where a host application can connect to multiple servers:

- **MCP Hosts**: Programs like Claude Desktop, IDEs, or AI tools that want to access data through MCP
- **MCP Clients**: Protocol clients that maintain 1:1 connections with servers
- **MCP Servers**: Lightweight programs that each expose specific capabilities through the standardized Model Context Protocol

GenAI toolbox acts as a server when running and uses MCP clients for each `mcp-server` source definition. It supports the `tools/list` and `tools/call` operations.

[mcp-spec]: https://modelcontextprotocol.io/introduction

## Requirements

### MCP Server

This source can be configured to a number of the MCP transports and specification versions.

## Example

```yaml
sources:
  my-mcp-server:
    kind: mcp-server
    endpoint: http://127.0.0.1:8080/mcp
    transport: http
    specVersion: 2025-03-26
    authMethod: bearer
    authSecret: ${MCP_SECRET}
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field**   | **type** | **required** | **description**                                    |
| ----------- | :------: | :----------: | -------------------------------------------------- |
| kind        |  string  |     true     | Must be "mcp-server".                              |
| endpoint    |  string  |     true     | Connect Uri - `http://127.0.0.1/mcp`               |
| specVersion |  string  |    false     | One of the supported mcp specification versions.   |
| transport   |  string  |    false     | One of the supported mcp transport types.          |
| authMethod  |  string  |    false     | One of the supported auth method types.            |
| authSecret  |  string  |    false     | The secret value used along with the `authMethod`. |

### Spec Versions

The supported MCP specification versions are the following:

- [`2024-11-05`][nov_2024]
- [`2025-03-26`][mar_2025]
- [`2025-06-18`][jun_2025]

_Note_: There may be some limitations for fully supporting each specification, but will only mature as the [go-sdk] improves.

> The default value is `2025-03-26`

[nov_2024]: https://modelcontextprotocol.io/specification/2024-11-05/basic/index
[mar_2025]: https://modelcontextprotocol.io/specification/2025-03-26/basic/index
[jun_2025]: https://modelcontextprotocol.io/specification/2025-06-18/basic/index
[go-sdk]: https://github.com/modelcontextprotocol/go-sdk/blob/aebd2449813d66cf742438ea37bdd8662fc10c30/mcp/streamable.go#L618

### Transports

The supported transports are the following:

- `stdio`
  - Support coming soon
  - Limitation: only one can be ran at a time
- `sse`
  - _Note_: is deprecated in the MCP spec
- `http`
  - The streamable http transport

> The default transport type is `http`

### Authentication

This source supports static authentication methods with a desired header:

- `none` - MCP servers without auth
- `apiKey` - Using the `X-API-KEY` header
- `bearer` - Using the `Authorization` header and adding the `Bearer $AUTHSECRET`

> The default authentication mode is `none` and the `authSecret` is required if the authentication mode is needed.
