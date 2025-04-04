---
title: "Connect via MCP Client"
type: docs
weight: 1
description: >
  How to connect to Toolbox from a MCP Client.
---

## Toolbox SDKs vs Model Context Protocol (MCP)
Toolbox now supports connections via both the native Toolbox SDKs and via [Model Context Protocol (MCP)](include link). However, Toolbox as several features which are not supported in the MCP specification (such as Authenticated Parameters and Authorized invocation). 

We recommend using the native SDKs over MCP clients to leverage these features. The native SDKs can be combined with MCP clients in many cases. 

### Protocol Versions
Toolbox currently supports the following versions of MCP specification:
* 2024-11-05

### Unavailable features when using MCP
Toolbox has several features that are not yet supported in the MCP specification:
* **AuthZ/AuthN:** There are no auth implementation in the `2024-11-05` specification. This includes:
  * [Authenticated Parameters](../resources/tools/_index.md#authenticated-parameters)
  * [Authorized Invocations](../resources/tools/_index.md#authorized-invocations)
* **Toolsets**: MCP does not have the concept of toolset. Hence, all tools are automatically loaded when using Toolbox with MCP.
* **Notifications:** Currently, editing Toolbox Tools requires a server restart. Clients should reload tools on disconnect to get the latest version. 


## Connecting to Toolbox with an MCP client
### Before you begin

{{< notice note >}} 
MCP is only compatible with Toolbox version 0.3.0 and above.
{{< /notice >}}

1. [Install](../getting-started/introduction/_index.md#installing-the-server) Toolbox version 0.3.0+.

1. Make sure you've set up and initialized your database.

1. [Set up](../getting-started/configure.md) your `tools.yaml` file.

### Connecting via HTTP
Toolbox supports the HTTP transport protocol with and without SSE.

{{< tabpane text=true >}} {{% tab header="HTTP with SSE" lang="en" %}}
Add the following configuration to your MCP client configuration:
```bash
{
  "mcpServers": {
    "toolbox": {
      "type": "sse",
      "url": "http://127.0.0.1:5000/mcp/sse",
    }
  }
}
```
{{% /tab %}} {{% tab header="HTTP POST" lang="en" %}}
Connect to Toolbox HTTP POST via `http://127.0.0.1:5000/mcp`.
{{% /tab %}} {{< /tabpane >}}

### Using the MCP Inspect with Toolbox

Use MCP [Inspector](https://github.com/modelcontextprotocol/inspector) for testing and debugging Toolbox server.

1. [Run Toolbox](../getting-started/introduction/_index.md#running-the-server).

1. In a separate terminal, run Inspector directly through `npx`:

    ```bash
    npx @modelcontextprotocol/inspector
    ```

1. For `Transport Type` dropdown menu, select `SSE`.

1. For `URL`, type in `http://127.0.0.1:5000/mcp/sse`.

1. Click the `Connect` button. Voila! You should be able to inspect your toolbox
   tools!