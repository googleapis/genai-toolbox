---
title: "PostgreSQL using MCP"
type: docs
weight: 2
description: >
  Connect your IDE to PostgreSQL using Toolbox.
---

[Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction) is an open protocol for connecting Large Language Models (LLMs) to data sources like Postgres. This guide covers how to use [MCP Toolbox for Databases][toolbox] to expose your developer assistant tools to a Postgres instance:

* [Cursor][cursor]
* [Windsurf][windsurf] (Codium)
* [Visual Studio Code ][vscode] (Copilot)
* [Cline][cline]  (VS Code extension)
* [Claude desktop][claudedesktop]
* [Claude code][claudecode]

[toolbox]: https://github.com/googleapis/genai-toolbox
[cursor]: #configure-your-mcp-client
[windsurf]: #configure-your-mcp-client
[vscode]: #configure-your-mcp-client
[cline]: #configure-your-mcp-client
[claudedesktop]: #configure-your-mcp-client
[claudecode]: #configure-your-mcp-client

{{< notice tip >}}
This guide can be used with [AlloyDB Omni](https://cloud.google.com/alloydb/omni/current/docs/overview).
{{< /notice >}}

## Set up the database

1. Create or select a PostgreSQL instance.

    * [Install PostgreSQL locally](https://www.postgresql.org/download/)
    * [Install AlloyDB Omni](https://cloud.google.com/alloydb/omni/current/docs/quickstart)

1. Create or reuse [a database user](https://cloud.google.com/alloydb/omni/current/docs/database-users/manage-users) and have the username and password ready.


## Install MCP Toolbox

1. Download the latest version of Toolbox as a binary. Select the [correct binary](https://github.com/googleapis/genai-toolbox/releases) corresponding to your OS and CPU architecture. You are required to use Toolbox version V0.5.0+:

   <!-- {x-release-please-start-version} -->
   {{< tabpane persist=header >}}
{{< tab header="linux/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.5.0/linux/amd64/toolbox
{{< /tab >}}

{{< tab header="darwin/arm64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.5.0/darwin/arm64/toolbox
{{< /tab >}}

{{< tab header="darwin/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.5.0/darwin/amd64/toolbox
{{< /tab >}}

{{< tab header="windows/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.5.0/windows/amd64/toolbox
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

## Configure and run Toolbox

The prebuild Toolbox binary will include the following tools:

1. **list_tables**: lists tables and descriptions
1. **execute_sql**: execute any SQL statement

To configure Toolbox, run the following steps:

1. Set the following environment variables:

    ```bash
    # The IP address of the Postgres instance.
    export POSTGRES_HOST="127.0.0.1"

    # The port of the Postgres instance.
    export POSTGRES_PORT=5432

    # The name of the database you want to connect to within the instance.
    export POSTGRES_DB="your-database-name"

    # The username for connecting to the database.
    export POSTGRES_USER="your-database-user"

    # The password for the specified database user.
    export POSTGRES_PASS="your-database-password"
    ```

## Configure your MCP Client

{{< tabpane text=true >}}
{{% tab header="Claude code" lang="en" %}}

1. Install [Claude Code](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code/overview).
1. Create a `.mcp.json` file in your project root if it doesn't exist.
1. Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "postgres": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","postgres","--stdio"],
          "env": [
            "POSTGRES_HOST": "",
            "POSTGRES_PORT": "",
            "POSTGRES_DB": "",
            "POSTGRES_USER": "",
            "POSTGRES_PASS": "",
          ],
        }
      }
    }
    ```

1. Restart Claude code to apply the new configuration.
{{% /tab %}}

{{% tab header="Claude desktop" lang="en" %}}

1. Install [`npx`](https://docs.npmjs.com/cli/v8/commands/npx).
1. Open [Claude desktop](https://claude.ai/download) and navigate to Settings.
1. Under the Developer tab, tap Edit Config to open the configuration file.
1. Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "postgres": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","postgres","--stdio"],
          "env": [
            "POSTGRES_HOST": "",
            "POSTGRES_PORT": "",
            "POSTGRES_DB": "",
            "POSTGRES_USER": "",
            "POSTGRES_PASS": "",
          ],
        }
      }
    }
    ```

1. Restart Claude desktop.
1. From the new chat screen, you should see a hammer (MCP) icon appear with the new MCP server available.
{{% /tab %}}

{{% tab header="Cline" lang="en" %}}

1. Open the [Cline](https://github.com/cline/cline) extension in VS Code and tap the **MCP Servers** icon.
1. Tap Configure MCP Servers to open the configuration file.
1. Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "postgres": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","postgres","--stdio"],
          "env": [
            "POSTGRES_HOST": "",
            "POSTGRES_PORT": "",
            "POSTGRES_DB": "",
            "POSTGRES_USER": "",
            "POSTGRES_PASS": "",
          ],
        }
      }
    }
    ```

1. You should see a green active status after the server is successfully connected.
{{% /tab %}}

{{% tab header="Cursor" lang="en" %}}

1. Create a `.cursor` directory in your project root if it doesn't exist.
1. Create a `.cursor/mcp.json` file if it doesn't exist and open it.
1. Add the following configuration:

    ```json
    {
      "mcpServers": {
        "postgres": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","postgres","--stdio"],
          "env": [
            "POSTGRES_HOST": "",
            "POSTGRES_PORT": "",
            "POSTGRES_DB": "",
            "POSTGRES_USER": "",
            "POSTGRES_PASS": "",
          ],
        }
      }
    }
    ```

1. [Cursor](https://www.cursor.com/) and navigate to **Settings > Cursor Settings > MCP**. You should see a green active status after the server is successfully connected.
{{% /tab %}}

{{% tab header="Visual Studio Code (Copilot)" lang="en" %}}

1. Open [VS Code](https://code.visualstudio.com/docs/copilot/overview) and create a `.vscode` directory in your project root if it doesn't exist.
1. Create a `.vscode/mcp.json` file if it doesn't exist and open it.
1. Add the following configuration:

    ```json
    {
      "mcpServers": {
        "postgres": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","postgres","--stdio"],
          "env": [
            "POSTGRES_HOST": "",
            "POSTGRES_PORT": "",
            "POSTGRES_DB": "",
            "POSTGRES_USER": "",
            "POSTGRES_PASS": "",
          ],
        }
      }
    }
    ```
{{% /tab %}}

{{% tab header="Windsurf" lang="en" %}}

1. Open [Windsurf](https://docs.codeium.com/windsurf) and navigate to the Cascade assistant.
1. Tap on the hammer (MCP) icon, then Configure to open the configuration file.
1. Add the following configuration:

    ```json
    {
      "mcpServers": {
        "postgres": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","postgres","--stdio"],
          "env": [
            "POSTGRES_HOST": "",
            "POSTGRES_PORT": "",
            "POSTGRES_DB": "",
            "POSTGRES_USER": "",
            "POSTGRES_PASS": "",
          ],
        }
      }
    }

    ```
{{% /tab %}}
{{< /tabpane >}}
