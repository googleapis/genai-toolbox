# Extension Version: 2026-07-28

This directory contains the schemas and specifications for the `2026-07-28` version of the experimental MCP Toolbox extension.

## Extension Identifier

The identifier we are using for this extension version is `com.google.cloud/toolbox.v1`.

**Currently Supported Capabilities:**

The `com.google.cloud/toolbox.v1` extension enables Toolbox-specific behaviors outside the standard MCP specification. By advertising this extension, the server signals that clients can leverage the following advanced features:

1. **Advanced Authentication Workflows:** Supports OAuth2/OIDC flows integrated directly into the tool parameters.

If this extension is disabled (e.g., via the `--disable-ext` CLI flag), the server will not advertise it during the discovery phase. This ensures connecting clients fall back to strictly standard MCP behavior, bypassing these proprietary capabilities.
