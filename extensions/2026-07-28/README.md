# Extension Version: 2026-07-28

This directory contains the schemas and specifications for the `2026-07-28` version of the experimental MCP Toolbox extension.

## Extension Identifier

The identifier we are using for this extension version is `com.google.cloud/toolbox.v1`.

## Supported Capabilities

- **Secure Parameters**: The secure parameter feature is strictly tied to the latest `v20260728` MCP protocol and the `com.google.cloud/toolbox.v1` experimental extension.

- **[Groups](./groups/specification/groups.md)**: Exposes a server's configured groups over MCP through the `groups/list` and `groups/get` methods.  
  Schema: [`groups/schema/schema.ts`](./groups/schema/schema.ts)
