---
title: "AlloyDB Admin API using MCP"
type: docs
weight: 2
description: >
  Create your AlloyDB database with MCP Toolbox.
---

This guide covers how to use [MCP Toolbox for Databases][toolbox] to create AlloyDB clusters and instances from IDE enabling their E2E journey.

- [Cursor][cursor]
- [Windsurf][windsurf] (Codium)
- [Visual Studio Code ][vscode] (Copilot)
- [Cline][cline] (VS Code extension)
- [Claude desktop][claudedesktop]
- [Claude code][claudecode]
- [Gemini CLI][geminicli]
- [Gemini Code Assist][geminicodeassist]

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

1. In the Google Cloud console, on the [project selector page](https://console.cloud.google.com/projectselector2/home/dashboard), select or create a Google Cloud project.

1. [Make sure that billing is enabled for your Google Cloud project](https://cloud.google.com/billing/docs/how-to/verify-billing-enabled#confirm_billing_is_enabled_on_a_project).

## Install MCP Toolbox

1. Download the latest version of Toolbox as a binary. Select the [correct binary](https://github.com/googleapis/genai-toolbox/releases) corresponding to your OS and CPU architecture. You are required to use Toolbox version V0.10.0+:

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

1. Install [Claude Code](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code/overview).
1. Create a `.mcp.json` file in your project root if it doesn't exist.
1. Generate Access token to be used as API_KEY using `gcloud auth print-access-token`.

    > **Note:** The lifetime of token is 1 hour.

1. Add the following configuration, replace the environment variables with your values, and save:

   ```json
   {
     "mcpServers": {
       "alloydb-admin": {
         "command": "./PATH/TO/toolbox",
         "args": ["--prebuilt", "alloydb-postgres-admin", "--stdio"],
         "env": {
           "API_KEY": "your-api-key"
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
1. Generate Access token to be used as API_KEY using `gcloud auth print-access-token`.

    > **Note:** The lifetime of token is 1 hour.

1. Add the following configuration, replace the environment variables with your values, and save:

   ```json
   {
     "mcpServers": {
       "alloydb-admin": {
         "command": "./PATH/TO/toolbox",
         "args": ["--prebuilt", "alloydb-postgres-admin", "--stdio"],
         "env": {
           "API_KEY": "your-api-key"
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
1. Generate Access token to be used as API_KEY using `gcloud auth print-access-token`.

    > **Note:** The lifetime of token is 1 hour.

1. Add the following configuration, replace the environment variables with your values, and save:

   ```json
   {
     "mcpServers": {
       "alloydb-admin": {
         "command": "./PATH/TO/toolbox",
         "args": ["--prebuilt", "alloydb-postgres-admin", "--stdio"],
         "env": {
           "API_KEY": "your-api-key"
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
1. Generate Access token to be used as API_KEY using `gcloud auth print-access-token`.

    > **Note:** The lifetime of token is 1 hour.

1. Add the following configuration, replace the environment variables with your values, and save:

   ```json
   {
     "mcpServers": {
       "alloydb-admin": {
         "command": "./PATH/TO/toolbox",
         "args": ["--prebuilt", "alloydb-postgres-admin", "--stdio"],
         "env": {
           "API_KEY": "your-api-key"
         }
       }
     }
   }
   ```

1. [Cursor](https://www.cursor.com/) and navigate to **Settings > Cursor Settings > MCP**. You should see a green active status after the server is successfully connected.
   {{% /tab %}}

{{% tab header="Visual Studio Code (Copilot)" lang="en" %}}

1.  Open [VS Code](https://code.visualstudio.com/docs/copilot/overview) and create a `.vscode` directory in your project root if it doesn't exist.
1.  Create a `.vscode/mcp.json` file if it doesn't exist and open it.
1.  Generate Access token to be used as API_KEY using `gcloud auth print-access-token`.

    > **Note:** The lifetime of token is 1 hour.

1.  Add the following configuration, replace the environment variables with your values, and save:

```json
{
  "mcpServers": {
    "alloydb-admin": {
      "command": "./PATH/TO/toolbox",
      "args": ["--prebuilt", "alloydb-postgres-admin", "--stdio"],
      "env": {
        "API_KEY": "your-api-key"
      }
    }
  }
}
```

{{% /tab %}}

{{% tab header="Windsurf" lang="en" %}}

1.  Open [Windsurf](https://docs.codeium.com/windsurf) and navigate to the Cascade assistant.
1.  Tap on the hammer (MCP) icon, then Configure to open the configuration file.
1.  Generate Access token to be used as API_KEY using `gcloud auth print-access-token`.

    > **Note:** The lifetime of token is 1 hour.

1.  Add the following configuration, replace the environment variables with your values, and save:

```json
{
  "mcpServers": {
    "alloydb-admin": {
      "command": "./PATH/TO/toolbox",
      "args": ["--prebuilt", "alloydb-postgres-admin", "--stdio"],
      "env": {
        "API_KEY": "your-api-key"
      }
    }
  }
}
```

{{% /tab %}}
{{% tab header="Gemini CLI" lang="en" %}}

1.  Install the [Gemini CLI](https://github.com/google-gemini/gemini-cli?tab=readme-ov-file#quickstart).
1.  In your working directory, create a folder named `.gemini`. Within it, create a `settings.json` file.
1.  Generate Access token to be used as API_KEY using `gcloud auth print-access-token`.

    > **Note:** The lifetime of token is 1 hour.

1.  Add the following configuration, replace the environment variables with your values, and save:

```json
{
  "mcpServers": {
    "alloydb-admin": {
      "command": "./PATH/TO/toolbox",
      "args": ["--prebuilt", "alloydb-postgres-admin", "--stdio"],
      "env": {
        "API_KEY": "your-api-key"
      }
    }
  }
}
```

{{% /tab %}}
{{% tab header="Gemini Code Assist" lang="en" %}}

1.  Install the [Gemini Code Assist](https://marketplace.visualstudio.com/items?itemName=Google.geminicodeassist) extension in Visual Studio Code.
1.  Enable Agent Mode in Gemini Code Assist chat.
1.  In your working directory, create a folder named `.gemini`. Within it, create a `settings.json` file.
1.  Generate Access token to be used as API_KEY using `gcloud auth print-access-token`.

    > **Note:** The lifetime of token is 1 hour.

1.  Add the following configuration, replace the environment variables with your values, and save:

```json
{
  "mcpServers": {
    "alloydb-admin": {
      "command": "./PATH/TO/toolbox",
      "args": ["--prebuilt", "alloydb-postgres-admin", "--stdio"],
      "env": {
        "API_KEY": "your-api-key"
      }
    }
  }
}
```

{{% /tab %}}
{{< /tabpane >}}

## Use Tools

Your AI tool is now connected to AlloyDB using MCP. Try asking your AI assistant to create a database, cluster or instance.

The following tools are available to the LLM:

1. **alloydb-create-cluster**: creates alloydb cluster
1. **alloydb-create-instance**: creates alloydb instance (PRIMARY, READ_POOL or SECONDARY)
1. **alloydb-get-operation**: polls on operations API until the operation is done.

{{< notice note >}}
Prebuilt tools are pre-1.0, so expect some tool changes between versions. LLMs will adapt to the tools available, so this shouldn't affect most users.
{{< /notice >}}

## Connect to your Data

After setting up an AlloyDB cluster and instance, you can [connect your IDE to the database](https://cloud.google.com/alloydb/docs/pre-built-tools-with-mcp-toolbox).
