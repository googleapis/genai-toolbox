# MCP Toolbox for Databases Server

The MCP Toolbox for Databases Server gives AI-powered development tools the ability to work with your custom tools. It is designed to simplify and secure the development of tools for interacting with databases.

# Prerequisites

*   Download and install [MCP Toolbox](https://github.com/googleapis/genai-toolbox):
    1.  **Download the Toolbox binary**:
        Download the latest binary for your operating system and architecture from the storage bucket. Check the [releases page](https://github.com/googleapis/genai-toolbox/releases) for additional versions: 
      
        <!-- {x-release-please-start-version} -->
        * To install Toolbox as a binary on Linux (AMD64):
          ```bash
          curl -L -o toolbox https://storage.googleapis.com/genai-toolbox/v0.21.0/linux/amd64/toolbox
          ```

        * To install Toolbox as a binary on macOS (Apple Silicon):
          ```bash
          curl -L -o toolbox https://storage.googleapis.com/genai-toolbox/v0.21.0/darwin/arm64/toolbox
          ```

        * To install Toolbox as a binary on macOS (Intel):
          ```bash
          curl -L -o toolbox https://storage.googleapis.com/genai-toolbox/v0.21.0/darwin/amd64/toolbox
          ```

        * To install Toolbox as a binary on Windows (AMD64):
          ```powershell
          curl -o toolbox.exe "https://storage.googleapis.com/genai-toolbox/v0.21.0/windows/amd64/toolbox.exe"
          ```
        <!-- {x-release-please-end} -->
        
    2.  **Make it executable**:

        ```bash
        chmod +x toolbox
        ```

    3.  **Add the binary to $PATH in `.~/bash_profile`** (Note: You may need to restart Antigravity for changes to take effect.):

        ```bash
        export PATH=$PATH:path/to/folder
        ```

        **On Windows, move binary to the `WindowsApps\` folder**:
        ```
        move "C:\Users\<path-to-binary>\toolbox.exe" "C:\Users\<username>\AppData\Local\Microsoft\WindowsApps\"
        ```
    
        **Tip:** Ensure the destination folder for your binary is included in
        your system's PATH environment variable. To check `PATH`, use `echo
        $PATH` (or `echo %PATH%` on Windows).

*   Any required APIs and permissions for connecting to your database.

> **Note:** If your database instance uses private IPs, you must run the MCP server in the same Virtual Private Cloud (VPC) network.

## Install & Configuration

1. In the Antigravity MCP Store, click the "Install" button.

2. Add your [`tools.yaml` configuration
file](https://googleapis.github.io/genai-toolbox/getting-started/configure/) to
the directory you are running Antigravity

## Usage

Interact with your custom tools using natural language.

## Custom MCP Server Configuration

```json
{
  "mcpServers": {
    "mcp-toolbox": {
      "command": "toolbox",
      "args": ["--tools-file", "you-tool-file.yaml"],
      "env": {
        "ENV_VAR_NAME": "ENV_VAR_VALUE",
      }
    }
  }
}
```

## Documentation

For more information, visit the [MCP Toolbox for Databases documentation](https://googleapis.github.io/genai-toolbox/getting-started/introduction/).
