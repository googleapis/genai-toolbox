# MCP Toolbox for Databases Server

The MCP Toolbox for Databases Server gives AI-powered development tools the ability to work with your custom tools. It is designed to simplify and secure the development of tools for interacting with databases.


## Prerequisites

*   [Node.js](https://nodejs.org/) installed.
*   A Google Cloud project with relevant APIs enabled.
*   Ensure [Application Default Credentials](https://cloud.google.com/docs/authentication/gcloud) are available in your environment.

## Install & Configuration

1.  In the Antigravity MCP Store, click the **Install** button. A configuration window will appear.

2.  Create your [`tools.yaml` configuration file](https://googleapis.github.io/genai-toolbox/getting-started/configure/).

3.  In the configuration window, enter the full absolute path to your `tools.yaml` file and click **Save**.

> [!NOTE]
> If you encounter issues with Windows Defender blocking the execution, you may need to configure an allowlist. See [Configure exclusions for Microsoft Defender Antivirus](https://learn.microsoft.com/en-us/microsoft-365/security/defender-endpoint/configure-exclusions-microsoft-defender-antivirus?view=o365-worldwide) for more details.

## Usage

Interact with your custom tools using natural language.

## Custom MCP Server Configuration

```json
{
  "mcpServers": {
    "mcp-toolbox": {
      "command": "npx",
      "args": ["-y", "@toolbox-sdk/server", "--tools-file", "your-tool-file.yaml"],
      "env": {
        "ENV_VAR_NAME": "ENV_VAR_VALUE",
      }
    }
  }
}
```

## Documentation

For more information, visit the [MCP Toolbox for Databases documentation](https://googleapis.github.io/genai-toolbox/getting-started/introduction/).
