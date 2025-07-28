---
title: "Looker using MCP"
type: docs
weight: 2
description: >
  Connect your IDE to Looker using Toolbox.
---

[Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction) is
an open protocol for connecting Large Language Models (LLMs) to data sources
like Postgres. This guide covers how to use [MCP Toolbox for Databases][toolbox]
to expose your developer assistant tools to a Looker instance:

* [Cursor][cursor]
* [Windsurf][windsurf] (Codium)
* [Visual Studio Code][vscode] (Copilot)
* [Cline][cline] (VS Code extension)
* [Claude desktop][claudedesktop]
* [Claude code][claudecode]

[toolbox]: https://github.com/googleapis/genai-toolbox
[cursor]: #configure-your-mcp-client
[windsurf]: #configure-your-mcp-client
[vscode]: #configure-your-mcp-client
[cline]: #configure-your-mcp-client
[claudedesktop]: #configure-your-mcp-client
[claudecode]: #configure-your-mcp-client

## Set up Looker

1. Get a Looker Client ID and Client Secret. Follow the directions
   [here](https://cloud.google.com/looker/docs/api-auth#authentication_with_an_sdk).

1. Have the base URL of your Looker instance available. It is likely
   something like `https://looker.example.com`. In some cases the API is
   listening at a different port, and you will need to use
   `https://looker.example.com:19999` instead.

## Install MCP Toolbox

1. Download the latest version of Toolbox as a binary. Select the [correct
   binary](https://github.com/googleapis/genai-toolbox/releases) corresponding
   to your OS and CPU architecture. You are required to use Toolbox version
   v0.10.0+:

   <!-- {x-release-please-start-version} -->
{{< tabpane persist=header >}}
{{< tab header="linux/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.10.0/linux/amd64/toolbox
{{< /tab >}}

{{< tab header="darwin/arm64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.10.0/darwin/arm64/toolbox
{{< /tab >}}

{{< tab header="darwin/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.10.0/darwin/amd64/toolbox
{{< /tab >}}

{{< tab header="windows/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.10.0/windows/amd64/toolbox.exe
{{< /tab >}}
{{< /tabpane >}}
    <!-- {x-release-please-end} -->

1. Make the binary executable:

    ```bash
    chmod +x toolbox
    ```

1. Verify the installation:

    ```bash
    ./toolbox --version
    ```

## Configure your MCP Client

{{< tabpane text=true >}}
{{% tab header="Claude code" lang="en" %}}

1. Install [Claude
   Code](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code/overview).
1. Create a `.mcp.json` file in your project root if it doesn't exist.
1. Add the following configuration, replace the environment variables with your
   values, and save:

    ```json
    {
      "mcpServers": {
        "looker-toolbox": {
          "command": "./PATH/TO/toolbox",
          "args": ["--stdio", "--prebuilt", "looker"],
          "env": {
            "LOOKER_BASE_URL": "https://looker.example.com",
            "LOOKER_CLIENT_ID": "",
            "LOOKER_CLIENT_SECRET": "",
            "LOOKER_VERIFY_SSL": "true"
          }
        }
      }
    }
    ```

1. Restart Claude Code to apply the new configuration.
{{% /tab %}}

{{% tab header="Claude desktop" lang="en" %}}

1. Open [Claude desktop](https://claude.ai/download) and navigate to Settings.
1. Under the Developer tab, tap Edit Config to open the configuration file.
1. Add the following configuration, replace the environment variables with your
   values, and save:

    ```json
    {
      "mcpServers": {
        "looker-toolbox": {
          "command": "./PATH/TO/toolbox",
          "args": ["--stdio", "--prebuilt", "looker"],
          "env": {
            "LOOKER_BASE_URL": "https://looker.example.com",
            "LOOKER_CLIENT_ID": "",
            "LOOKER_CLIENT_SECRET": "",
            "LOOKER_VERIFY_SSL": "true"
          }
        }
      }
    }
    ```

1. Restart Claude desktop.
1. From the new chat screen, you should see a hammer (MCP) icon appear with the
   new MCP server available.
{{% /tab %}}

{{% tab header="Cline" lang="en" %}}

1. Open the [Cline](https://github.com/cline/cline) extension in VS Code and tap
   the **MCP Servers** icon.
1. Tap Configure MCP Servers to open the configuration file.
1. Add the following configuration, replace the environment variables with your
   values, and save:

    ```json
    {
      "mcpServers": {
        "looker-toolbox": {
          "command": "./PATH/TO/toolbox",
          "args": ["--stdio", "--prebuilt", "looker"],
          "env": {
            "LOOKER_BASE_URL": "https://looker.example.com",
            "LOOKER_CLIENT_ID": "",
            "LOOKER_CLIENT_SECRET": "",
            "LOOKER_VERIFY_SSL": "true"
          }
        }
      }
    }
    ```

1. You should see a green active status after the server is successfully
   connected.
{{% /tab %}}

{{% tab header="Cursor" lang="en" %}}

1. Create a `.cursor` directory in your project root if it doesn't exist.
1. Create a `.cursor/mcp.json` file if it doesn't exist and open it.
1. Add the following configuration, replace the environment variables with your
   values, and save:

    ```json
    {
      "mcpServers": {
        "looker-toolbox": {
          "command": "./PATH/TO/toolbox",
          "args": ["--stdio", "--prebuilt", "looker"],
          "env": {
            "LOOKER_BASE_URL": "https://looker.example.com",
            "LOOKER_CLIENT_ID": "",
            "LOOKER_CLIENT_SECRET": "",
            "LOOKER_VERIFY_SSL": "true"
          }
        }
      }
    }
    ```

1. Open [Cursor](https://www.cursor.com/) and navigate to **Settings > Cursor
   Settings > MCP**. You should see a green active status after the server is
   successfully connected.
{{% /tab %}}

{{% tab header="Visual Studio Code (Copilot)" lang="en" %}}

1. Open [VS Code](https://code.visualstudio.com/docs/copilot/overview) and
   create a `.vscode` directory in your project root if it doesn't exist.
1. Create a `.vscode/mcp.json` file if it doesn't exist and open it.
1. Add the following configuration, replace the environment variables with your
   values, and save:

    ```json
    {
      "mcpServers": {
        "looker-toolbox": {
          "command": "./PATH/TO/toolbox",
          "args": ["--stdio", "--prebuilt", "looker"],
          "env": {
            "LOOKER_BASE_URL": "https://looker.example.com",
            "LOOKER_CLIENT_ID": "",
            "LOOKER_CLIENT_SECRET": "",
            "LOOKER_VERIFY_SSL": "true"
          }
        }
      }
    }
    ```

{{% /tab %}}

{{% tab header="Windsurf" lang="en" %}}

1. Open [Windsurf](https://docs.codeium.com/windsurf) and navigate to the
   Cascade assistant.
1. Tap on the hammer (MCP) icon, then Configure to open the configuration file.
1. Add the following configuration, replace the environment variables with your
   values, and save:

    ```json
    {
      "mcpServers": {
        "looker-toolbox": {
          "command": "./PATH/TO/toolbox",
          "args": ["--stdio", "--prebuilt", "looker"],
          "env": {
            "LOOKER_BASE_URL": "https://looker.example.com",
            "LOOKER_CLIENT_ID": "",
            "LOOKER_CLIENT_SECRET": "",
            "LOOKER_VERIFY_SSL": "true"
          }
        }
      }
    }

    ```

{{% /tab %}}
{{< /tabpane >}}

## Use Tools

Your AI tool is now connected to Looker using MCP. Try asking your AI
assistant to list models, explores, dimensions, and measures. Run a
query, retrieve the SQL for a query, and run a saved Look.

The following tools are available to the LLM:

1. **get_models**: list the LookML models in Looker
1. **get_explores**: list the explores in a given model
1. **get_dimensions**: list the dimensions in a given explore
1. **get_measures**: list the measures in a given explore
1. **get_filters**: list the filters in a given explore
1. **get_parameters**: list the parameters in a given explore
1. **query**: Run a query
1. **query_sql**: Return the SQL generated by Looker for a query
1. **get_looks**: Return the saved Looks that match a title or description
1. **run_look**: Run a saved Look and return the data

{{< notice note >}}
Prebuilt tools are pre-1.0, so expect some tool changes between versions. LLMs
will adapt to the tools available, so this shouldn't affect most users.
{{< /notice >}}
