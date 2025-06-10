---
title: "Cloud SQL for MySQL using MCP"
type: docs
weight: 2
description: >
  Connect your IDE to Cloud SQL for MySQL using Toolbox.
---

[Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction) is an open protocol for connecting Large Language Models (LLMs) to data sources like Cloud SQL. This guide covers how to use [MCP Toolbox for Databases][toolbox] to expose your developer assistant tools to a Cloud SQL for MySQL instance:

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

## Before you begin

1. In the Google Cloud console, on the [project selector page](https://console.cloud.google.com/projectselector2/home/dashboard), select or create a Google Cloud project.

1. [Make sure that billing is enabled for your Google Cloud project](https://cloud.google.com/billing/docs/how-to/verify-billing-enabled#confirm_billing_is_enabled_on_a_project).


## Set up the database

1. [Enable the Cloud SQL Admin API in the Google Cloud project](https://console.cloud.google.com/flows/enableapi?apiid=sqladmin&redirect=https://console.cloud.google.com).

1. [Create a Cloud SQL for MySQL instance](https://cloud.google.com/sql/docs/mysql/create-instance). These instructions assume that your Cloud SQL instance has a [public IP address](https://cloud.google.com/sql/docs/mysql/configure-ip). By default, Cloud SQL assigns a public IP address to a new instance. Toolbox will connect securely using the [Cloud SQL connectors](https://cloud.google.com/sql/docs/mysql/language-connectors).

1. Configure the required roles and permissions to complete this task. You will need [Cloud SQL > Client](https://cloud.google.com/sql/docs/mysql/roles-and-permissions#proxy-roles-permissions) role (`roles/cloudsql.client`) or equivalent IAM permissions to connect to the instance.

1. Configured [Application Default Credentials (ADC)](https://cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) for your environment.

1. Create or reuse [a database user](https://cloud.google.com/sql/docs/mysql/create-manage-users) and have the username and password ready.


## Install MCP Toolbox

1. Download the latest version of Toolbox as a binary. Select the [correct binary](https://github.com/googleapis/genai-toolbox/releases) corresponding to your OS and CPU architecture. You are required to use Toolbox version V0.6.0+:

   <!-- {x-release-please-start-version} -->
   {{< tabpane persist=header >}}
{{< tab header="linux/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.7.0/linux/amd64/toolbox
{{< /tab >}}

{{< tab header="darwin/arm64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.7.0/darwin/arm64/toolbox
{{< /tab >}}

{{< tab header="darwin/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.7.0/darwin/amd64/toolbox
{{< /tab >}}

{{< tab header="windows/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.7.0/windows/amd64/toolbox
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

1. Install [Claude Code](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code/overview).
1. Create a `.mcp.json` file in your project root if it doesn't exist.
1. Add the following configuration, replace the environment variables with your values, and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-mysql": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","cloud-sql-mysql","--stdio"],
          "env": {
            "CLOUD_SQL_MYSQL_PROJECT": "",
            "CLOUD_SQL_MYSQL_REGION": "",
            "CLOUD_SQL_MYSQL_INSTANCE": "",
            "CLOUD_SQL_MYSQL_DATABASE": "",
            "CLOUD_SQL_MYSQL_USER": "",
            "CLOUD_SQL_MYSQL_PASSWORD": ""
          }
        }
      }
    }
    ```

1. Restart Claude code to apply the new configuration.
{{% /tab %}}

{{% tab header="Claude desktop" lang="en" %}}

1. Open [Claude desktop](https://claude.ai/download) and navigate to Settings.
1. Under the Developer tab, tap Edit Config to open the configuration file.
1. Add the following configuration, replace the environment variables with your values, and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-mysql": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","cloud-sql-mysql","--stdio"],
          "env": {
            "CLOUD_SQL_MYSQL_PROJECT": "",
            "CLOUD_SQL_MYSQL_REGION": "",
            "CLOUD_SQL_MYSQL_INSTANCE": "",
            "CLOUD_SQL_MYSQL_DATABASE": "",
            "CLOUD_SQL_MYSQL_USER": "",
            "CLOUD_SQL_MYSQL_PASSWORD": ""
          }
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
1. Add the following configuration, replace the environment variables with your values, and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-mysql": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","cloud-sql-mysql","--stdio"],
          "env": {
            "CLOUD_SQL_MYSQL_PROJECT": "",
            "CLOUD_SQL_MYSQL_REGION": "",
            "CLOUD_SQL_MYSQL_INSTANCE": "",
            "CLOUD_SQL_MYSQL_DATABASE": "",
            "CLOUD_SQL_MYSQL_USER": "",
            "CLOUD_SQL_MYSQL_PASSWORD": ""
          }
        }
      }
    }
    ```

1. You should see a green active status after the server is successfully connected.
{{% /tab %}}

{{% tab header="Cursor" lang="en" %}}

1. Create a `.cursor` directory in your project root if it doesn't exist.
1. Create a `.cursor/mcp.json` file if it doesn't exist and open it.
1. Add the following configuration, replace the environment variables with your values, and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-mysql": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","cloud-sql-mysql","--stdio"],
          "env": {
            "CLOUD_SQL_MYSQL_PROJECT": "",
            "CLOUD_SQL_MYSQL_REGION": "",
            "CLOUD_SQL_MYSQL_INSTANCE": "",
            "CLOUD_SQL_MYSQL_DATABASE": "",
            "CLOUD_SQL_MYSQL_USER": "",
            "CLOUD_SQL_MYSQL_PASSWORD": ""
          }
      }
    }
    ```

1. [Cursor](https://www.cursor.com/) and navigate to **Settings > Cursor Settings > MCP**. You should see a green active status after the server is successfully connected.
{{% /tab %}}

{{% tab header="Visual Studio Code (Copilot)" lang="en" %}}

1. Open [VS Code](https://code.visualstudio.com/docs/copilot/overview) and create a `.vscode` directory in your project root if it doesn't exist.
1. Create a `.vscode/mcp.json` file if it doesn't exist and open it.
1. Add the following configuration, replace the environment variables with your values, and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-mysql": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","cloud-sql-mysql","--stdio"],
          "env": {
            "CLOUD_SQL_MYSQL_PROJECT": "",
            "CLOUD_SQL_MYSQL_REGION": "",
            "CLOUD_SQL_MYSQL_INSTANCE": "",
            "CLOUD_SQL_MYSQL_DATABASE": "",
            "CLOUD_SQL_MYSQL_USER": "",
            "CLOUD_SQL_MYSQL_PASSWORD": ""
          }
        }
      }
    }
    ```
{{% /tab %}}

{{% tab header="Windsurf" lang="en" %}}

1. Open [Windsurf](https://docs.codeium.com/windsurf) and navigate to the Cascade assistant.
1. Tap on the hammer (MCP) icon, then Configure to open the configuration file.
1. Add the following configuration, replace the environment variables with your values, and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-mysql": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","cloud-sql-mysql","--stdio"],
          "env": {
            "CLOUD_SQL_MYSQL_PROJECT": "",
            "CLOUD_SQL_MYSQL_REGION": "",
            "CLOUD_SQL_MYSQL_INSTANCE": "",
            "CLOUD_SQL_MYSQL_DATABASE": "",
            "CLOUD_SQL_MYSQL_USER": "",
            "CLOUD_SQL_MYSQL_PASSWORD": ""
          }
        }
      }
    }

    ```
{{% /tab %}}
{{< /tabpane >}}

## Use Tools

Your AI tool is now connected to Cloud SQL for MySQL using MCP. Try asking your AI assistant to list tables, create a table, or define and execute other SQL statements.

The following tools are available to the LLM:

1. **list_tables**: lists tables and descriptions
1. **execute_sql**: execute any SQL statement

{{< notice note >}}
Prebuilt tools are pre-1.0, so expect some tool changes between versions. LLMs will adapt to the tools available, so this shouldn't affect most users.
{{< /notice >}}
