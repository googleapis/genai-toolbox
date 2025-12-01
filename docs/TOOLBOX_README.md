# MCP Toolbox for Databases Server

The MCP Toolbox for Databases Server gives AI-powered development tools the ability to work with your custom tools. It is designed to simplify and secure the development of tools for interacting with databases.


## Install & Configuration

1. Add your [`tools.yaml` configuration
file](https://googleapis.github.io/genai-toolbox/getting-started/configure/) to
the directory you are running Antigravity.

2. In the Antigravity MCP Store, click the "Install" button.

## Usage

Interact with your custom tools using natural language.

## Custom MCP Server Configuration

```json
{
  "mcpServers": {
    "mcp-toolbox": {
      "command": "npx",
      "args": ["@toolbox-sdk/server", "--tools-file", "your-tool-file.yaml"],
      "env": {
        "ENV_VAR_NAME": "ENV_VAR_VALUE",
      }
    }
  }
}
```

## Documentation

For more information, visit the [MCP Toolbox for Databases documentation](https://googleapis.github.io/genai-toolbox/getting-started/introduction/).
