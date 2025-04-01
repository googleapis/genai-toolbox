---
title: "Connect via MCP Client"
type: docs
weight: 1
description: >
  How to connect to Toolbox from MCP Client.
---

## Toolbox support for MCP
To ensure a seamless compatibility between MCP and Toolbox, we provide a native
integration of MCP.

### Protocol Versions
Toolbox currently supports the following versions of MCP specification:
* 2024-11-05

### What do we not support with Toolbox MCP
There are certain features within the MCP specification that we are not/not yet
supporting:
* Auth: There are no auth implementation in the `2024-11-05` specification.
* Toolsets: MCP does not have the concept of toolset. Hence, all tools are
  automatically loaded when using Toolbox with MCP.
* Notifications: We do not provide list changed notifications for Tools.
* Optional Parameters: We have found that LLM performs better without optional
  parameter, hence it is not currently supported in the beta version of Toolbox.
  All parameters are treated as required.


## Using Toolbox with MCP Client
### Before you begin

{{< notice note >}} 
MCP is only compatible with Toolbox version 0.3.0 and above.
{{< /notice >}}

1. [Install](../getting-started/introduction/_index.md#installing-the-server) Toolbox version 0.3.0+.

1. Make sure you've set up and initialized your database.

1. Set up your `tools.yaml` file.

### Connecting via HTTP (SSE)

To connect with MCP client that supports HTTP transport with SSE, add the following configuration to your MCP client configuration:
```bash
{
  "mcpServers": {
    "toolbox": {
      "type": "sse",
      "url": "https://127.0.0.1:5000/mcp/sse",
    }
  }
}

```

### Connecting via HTTP

To connect with MCP client that support HTTP transport without SSE, you can
connect via `https://127.0.0.1:5000/mcp`.


### Debug your Toolbox with the MCP Inspector

Use MCP Inspector for testing and debugging Toolbox server.

1. [Run Toolbox](../getting-started/introduction/_index.md#running-the-server).

1. In a separate terminal, run Inspector directly through `npx`:

    ```bash
    npx @modelcontextprotocol/inspector
    ```

1. For `Transport Type` dropdown menu, select `SSE`.

1. For `URL`, type in `http://127.0.0.1:5000/mcp/sse`.

1. Click the `Connect` button. Voila! You should be able to inspect your toolbox
   tools!