---
title: "Using Model Context Protocol (MCP)"
type: docs
weight: 2
description: >
  Connect your AI developer assistant tools to Toolbox using MCP.
---

[Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction) is an open protocol for connecting Large Language Models (LLMs) to data sources like Cloud SQL. This guide covers how to use [MCP Toolbox for Databases][toolbox] to expose your developer assistant tools to a Cloud SQL for Postgres instance:

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

## Before you begin:

1. In the Google Cloud console, on the [project selector page](https://console.cloud.google.com/projectselector2/home/dashboard), select or create a Google Cloud project.

1. [Make sure that billing is enabled for your Google Cloud project](https://cloud.google.com/billing/docs/how-to/verify-billing-enabled#confirm_billing_is_enabled_on_a_project).


## Set up a Cloud SQL instance

1. [Enable the Cloud SQL Admin API in the Google Cloud project](https://console.cloud.google.com/flows/enableapi?apiid=sqladmin&redirect=https://console.cloud.google.com).

1. [Create a Cloud SQL for PostgreSQL instance](https://cloud.google.com/sql/docs/postgres/create-instance). These instructions assume that your Cloud SQL instance has a [public IP address](https://cloud.google.com/sql/docs/postgres/configure-ip). By default, Cloud SQL assigns a public IP address to a new instance. Toolbox will connect securely using the [Cloud SQL connectors](https://cloud.google.com/sql/docs/postgres/language-connectors).

1. Configure the required roles and permissions to complete this task. You will need [Cloud SQL > Client](https://cloud.google.com/sql/docs/postgres/roles-and-permissions#proxy-roles-permissions) role (`roles/cloudsql.client`) or equivalent IAM permissions to connect to the instance.

1. Configured [Application Default Credentials (ADC)](https://cloud.google.com/docs/authentication/set-up-adc-local-dev-environment) for your environment.

1. Create or reuse [a database user](https://cloud.google.com/sql/docs/postgres/create-manage-users) and have the username and password ready.


## Install MCP Toolbox

1. Download the latest version of Toolbox as a binary:

    {{< notice tip >}}
    Select the
    [correct binary](https://github.com/googleapis/genai-toolbox/releases)
    corresponding to your OS and CPU architecture.
    {{< /notice >}}

    {{< tabpane persist=header >}}
    {{% tab header="linux/amd64" lang="bash" %}}
      <!-- {x-release-please-start-version} -->
      curl -O https://storage.googleapis.com/genai-toolbox/v0.4.0/linux/amd64/toolbox
      <!-- {x-release-please-end} -->
    {{< /tab >}}

    {{% tab header="darwin/arm64" lang="bash" %}}
      <!-- {x-release-please-start-version} -->
      curl -O https://storage.googleapis.com/genai-toolbox/v0.4.0/darwin/arm64/toolbox
      <!-- {x-release-please-end} -->
    {{< /tab >}}

    {{% tab header="darwin/amd64" lang="bash" %}}
      <!-- {x-release-please-start-version} -->
      curl -O https://storage.googleapis.com/genai-toolbox/v0.4.0/darwin/amd64/toolbox
      <!-- {x-release-please-end} -->
    {{< /tab >}}

    {{% tab header="windows/amd64" lang="bash" %}}
      <!-- {x-release-please-start-version} -->
      curl -O https://storage.googleapis.com/genai-toolbox/v0.4.0/windows/amd64/toolbox
      <!-- {x-release-please-end} -->
    {{< /tab >}}
    {{< /tabpane >}}

1. Make the binary executable:

    ```bash
    chmod +x toolbox
    ```

3. Verify the installation:

    ```bash
    ./toolbox --version
    ```

## Configure and run Toolbox

This section will create a `tools.yaml` file, which will define which tools your AI Agent will have access to. You can add, remove, or edit tools as needed to make sure you have the best tools for your workflows.

This will configure the following tools:
<!-- TODO: update -->
1. **list_tables**: lists all tables in your PostgreSQL instance
2. **list_schema:** Lists the schema for a particular db table
3. **execute_sql**: execute queries to retrieve data from your database.
4. **create_table**: create a new table in your database.
5. **drop_table**: drop an existing table in your database.

To configure Toolbox, run the following steps:

1. Set the following environment variables:

    ```bash
    # The ID of your Google Cloud Project where the Cloud SQL instance is located.
    export CLOUD_SQL_PROJECT="your-gcp-project-id"

    # The region where your Cloud SQL instance is located (e.g., us-central1).
    export CLOUD_SQL_REGION="your-instance-region"

    # The name of your Cloud SQL instance.
    export CLOUD_SQL_INSTANCE="your-instance-name"

    # The name of the database you want to connect to within the instance.
    export CLOUD_SQL_DB="your-database-name"

    # The username for connecting to the database.
    export CLOUD_SQL_USER="your-database-user"

    # The password for the specified database user.
    export CLOUD_SQL_PASS="your-database-password"
    ```

2. Create a `tools.yaml` file (`touch tools.yaml`)

3. Copy and paste the following contents into the `tools.yaml`:

    ```yaml
    sources:
      my-cloud-sql-pg-source:
        kind: cloud-sql-postgres
        project: ${CLOUD_SQL_PROJECT}
        region: ${CLOUD_SQL_REGION}
        instance: ${CLOUD_SQL_INSTANCE}
        database: ${CLOUD_SQL_DB}
        user: ${CLOUD_SQL_USER}
        password: ${CLOUD_SQL_PASS}
    tools:
      TODO - TOOLS GO HERE
    ```

4. Start Toolbox to listen on `127.0.0.1:5000`:

    ```bash
    ./toolbox --tools-file tools.yaml --address 127.0.0.1 --port 5000
    ```

{{< notice tip >}}
To stop the Toolbox server when you're finished, press `ctrl+c` to send the terminate signal.
{{< /notice >}}

## Configure your MCP Client
{{< tabpane text=true >}}
{{% tab header="Claude code" lang="en" %}}

1. Install [Claude Code](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code/overview).
2. Create a `.mcp.json` file in your project root if it doesn't exist.
3. Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres": {
          "type": "sse",
          "url": "http://127.0.0.1:5000/mcp/sse",
        }
      }
    }
    ```

4. Restart Claude code to apply the new configuration.
{{< /tab >}}

{{% tab header="Claude desktop" lang="en" %}}

1. Open [Claude desktop](https://claude.ai/download) and navigate to Settings.
2. Under the Developer tab, tap Edit Config to open the configuration file.
3. Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres": {
          "type": "sse",
          "url": "http://127.0.0.1:5000/mcp/sse",
        }
      }
    }
    ```

4. Restart Claude desktop.
5. From the new chat screen, you should see a hammer (MCP) icon appear with the new MCP server available.
{{< /tab >}}

{{% tab header="Cline" lang="en" %}}

1. Open the [Cline](https://github.com/cline/cline) extension in VS Code and tap the **MCP Servers** icon.
2. Tap Configure MCP Servers to open the configuration file.
3. Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres": {
          "type": "sse",
          "url": "http://127.0.0.1:5000/mcp/sse",
        }
      }
    }
    ```

4. You should see a green active status after the server is successfully connected.
{{< /tab >}}

{{% tab header="Cursor" lang="en" %}}

1. Open [Cursor](https://www.cursor.com/) and create a `.cursor` directory in your project root if it doesn't exist.
2. Create a `.cursor/mcp.json` file if it doesn't exist and open it.
3. Add the following configuration:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres": {
          "type": "sse",
          "url": "http://127.0.0.1:5000/mcp/sse",
        }
      }
    }
    ```

4. Open Cursor and navigate to **Settings/MCP**. You should see a green active status after the server is successfully connected.
{{< /tab >}}

{{% tab header="Visual Studio Code (Copilot)" lang="en" %}}

1. Open [VS Code](https://code.visualstudio.com/docs/copilot/overview) and create a `.vscode` directory in your project root if it doesn't exist.
2. Create a `.vscode/mcp.json` file if it doesn't exist and open it.
3. Add the following configuration:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres": {
          "type": "sse",
          "url": "http://127.0.0.1:5000/mcp/sse",
        }
      }
    }
    ```
{{< /tab >}}

{{% tab header="Windsurf" lang="en" %}}

1. Open [Windsurf](https://docs.codeium.com/windsurf) and navigate to the Cascade assistant.
2. Tap on the hammer (MCP) icon, then Configure to open the configuration file.
3. Add the following configuration:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres": {
          "type": "sse",
          "url": "http://127.0.0.1:5000/mcp/sse",
        }
      }
    }
    ```
{{< /tab >}}
{{< /tabpane >}}