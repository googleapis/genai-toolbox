# Secure Parameters Extension Specification

This document outlines the specification for the `secureParams` extension for internal developers and core team contributors.

## Overview

This extension adds support for secure parameters under the DRAFT-2026-v1 (vdraft) MCP specification. 
On the wire format, a tool manifest can now include an optional `secureInputSchema` alongside the standard `inputSchema`. 
The server validates that sensitive arguments defined in the `secureInputSchema` are exclusively passed via the `secureArguments` object rather than standard `arguments` or URL-bound parameters.

## Capabilities
Clients that support this extension should advertise the following capability in their initialization request:

```json
{
  "experimental": {
    "com.google.cloud/secure-params": true
  }
}
```

## Tool Execution

When calling a tool with secure parameters, the sensitive values must be provided in the `secureArguments` object instead of the regular `arguments` object.

```json
{
  "method": "tools/call",
  "params": {
    "name": "secure_tool",
    "arguments": {
      "query": "hello"
    },
    "secureArguments": {
      "api_key": "secret"
    }
  }
}
```
