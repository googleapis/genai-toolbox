---
title: "Cloud SQL for PostgreSQL Admin using MCP"
type: docs
weight: 3
description: >
  Create and manage Cloud SQL for PostgreSQL (Admin) using Toolbox.
---

This guide covers how to use [MCP Toolbox for Databases][toolbox] to expose your
developer assistant tools to create and manage Cloud SQL for PostgreSQL
instance, database and users:

* [Cursor][cursor]
* [Windsurf][windsurf] (Codium)
* [Visual Studio Code][vscode] (Copilot)
* [Cline][cline]  (VS Code extension)
* [Claude desktop][claudedesktop]
* [Claude code][claudecode]
* [Gemini CLI][geminicli]
* [Gemini Code Assist][geminicodeassist]

[toolbox]: https://github.com/googleapis/genai-toolbox
[cursor]: #configure-your-mcp-client
[windsurf]: #configure-your-mcp-client
[vscode]: #configure-your-mcp-client
[cline]: #configure-your-mcp-client
[claudedesktop]: #configure-your-mcp-client
[claudecode]: #configure-your-mcp-client
[geminicli]: #configure-your-mcp-client
[geminicodeassist]: #configure-your-mcp-client

## Before you begin

1. In the Google Cloud console, on the [project selector
   page](https://console.cloud.google.com/projectselector2/home/dashboard),
   select or create a Google Cloud project.

1. [Make sure that billing is enabled for your Google Cloud
   project](https://cloud.google.com/billing/docs/how-to/verify-billing-enabled#confirm_billing_is_enabled_on_a_project).

1. Grant the necessary IAM roles to the user that will be running the MCP
   server. The tools available will depend on the roles granted:
    * `roles/cloudsql.viewer`: Provides read-only access to resources.
        * `get_instance`
        * `list_instances`
        * `list_databases`
        * `wait_for_operation`
    * `roles/cloudsql.editor`: Provides permissions to manage existing resources.
        * All `viewer` tools
        * `create_database`
    * `roles/cloudsql.admin`: Provides full control over all resources.
        * All `editor` and `viewer` tools
        * `create_instance`
        * `create_user`
        * `clone_instance`

## Install MCP Toolbox

1. Download the latest version of Toolbox as a binary. Select the [correct
   binary](https://github.com/googleapis/genai-toolbox/releases) corresponding
   to your OS and CPU architecture. You are required to use Toolbox version
   V0.15.0+:

   <!-- {x-release-please-start-version} -->
   {{< tabpane persist=header >}}
{{< tab header="linux/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.15.0/linux/amd64/toolbox
{{< /tab >}}

{{< tab header="darwin/arm64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.15.0/darwin/arm64/toolbox
{{< /tab >}}

{{< tab header="darwin/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.15.0/darwin/amd64/toolbox
{{< /tab >}}

{{< tab header="windows/amd64" lang="bash" >}}
curl -O https://storage.googleapis.com/genai-toolbox/v0.15.0/windows/amd64/toolbox.exe
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
1. Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres-admin": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","cloud-sql-postgres-admin","--stdio"],
          "env": {
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
1. Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres-admin": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","cloud-sql-postgres-admin","--stdio"],
          "env": {
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
1. Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres-admin": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","cloud-sql-postgres-admin","--stdio"],
          "env": {
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
1. Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres-admin": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","cloud-sql-postgres-admin","--stdio"],
          "env": {
          }
        }
      }
    }
    ```

1. [Cursor](https://www.cursor.com/) and navigate to **Settings > Cursor
   Settings > MCP**. You should see a green active status after the server is
   successfully connected.
{{% /tab %}}

{{% tab header="Visual Studio Code (Copilot)" lang="en" %}}

1. Open [VS Code](https://code.visualstudio.com/docs/copilot/overview) and
   create a `.vscode` directory in your project root if it doesn't exist.
1. Create a `.vscode/mcp.json` file if it doesn't exist and open it.
1. Add the following configuration and save:

    ```json
    {
      "servers": {
        "cloud-sql-postgres-admin": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","cloud-sql-postgres-admin","--stdio"],
          "env": {
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
1. Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres-admin": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","cloud-sql-postgres-admin","--stdio"],
          "env": {
          }
        }
      }
    }
    ```

{{% /tab %}}

{{% tab header="Gemini CLI" lang="en" %}}

1.  Install the [Gemini
    CLI](https://github.com/google-gemini/gemini-cli?tab=readme-ov-file#quickstart).
1.  In your working directory, create a folder named `.gemini`. Within it,
    create a `settings.json` file.
1.  Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres-admin": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","cloud-sql-postgres-admin","--stdio"],
          "env": {
          }
        }
      }
    }
    ```

{{% /tab %}}

{{% tab header="Gemini Code Assist" lang="en" %}}

1.  Install the [Gemini Code
    Assist](https://marketplace.visualstudio.com/items?itemName=Google.geminicodeassist)
    extension in Visual Studio Code.
1.  Enable Agent Mode in Gemini Code Assist chat.
1.  In your working directory, create a folder named `.gemini`. Within it,
    create a `settings.json` file.
1.  Add the following configuration and save:

    ```json
    {
      "mcpServers": {
        "cloud-sql-postgres-admin": {
          "command": "./PATH/TO/toolbox",
          "args": ["--prebuilt","cloud-sql-postgres-admin","--stdio"],
          "env": {
          }
        }
      }
    }
    ```

{{% /tab %}}
{{< /tabpane >}}

## Use Tools

Your AI tool is now connected to Cloud SQL for PostgreSQL using MCP.

The `cloud-sql-postgres-admin` server provides tools for managing your Cloud SQL
instances and interacting with your database:

* **create_instance**: Creates a new Cloud SQL for PostgreSQL instance.
* **get_instance**: Gets information about a Cloud SQL instance.
* **list_instances**: Lists Cloud SQL instances in a project.
* **create_database**: Creates a new database in a Cloud SQL instance.
* **list_databases**: Lists all databases for a Cloud SQL instance.
* **create_user**: Creates a new user in a Cloud SQL instance.
* **wait_for_operation**: Waits for a Cloud SQL operation to complete.
* **clone_instance**: Creates a clone of an existing Cloud SQL for PostgreSQL instance.

{{< notice note >}}
Prebuilt tools are pre-1.0, so expect some tool changes between versions. LLMs
will adapt to the tools available, so this shouldn't affect most users.
{{< /notice >}}
