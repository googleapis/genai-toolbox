---
title: "ADK"
type: docs
weight: 1
description: >
  MCP Toolbox SDK for integrating functionalities of MCP Toolbox into your ADK apps.
---

## Overview

The `toolbox-adk` package provides a Python interface to the MCP Toolbox service, enabling you to load and invoke tools from your own applications.

## Installation

```bash
pip install google-adk[toolbox]
```

## Usage

The primary entry point is the `ToolboxToolset`, which loads tools from a remote Toolbox server and adapts them for use with ADK agents.

{{< notice note>}}
This package contains the core implementation of the `ToolboxToolset`. The `ToolboxToolset` provided in the [`google-adk`](https://github.com/google/adk-python/blob/758d337c76d877e3174c35f06551cc9beb1def06/src/google/adk/tools/toolbox_toolset.py#L35) package is a shim that simply delegates all functionality to this implementation.
{{< /notice >}}

```python
from google.adk.tools.toolbox_toolset import ToolboxToolset
from google.adk.agents import Agent

# Create the Toolset
toolset = ToolboxToolset(
    server_url="http://127.0.0.1:5000" 
)

# Use in your ADK Agent
agent = Agent(tools=[toolset])
```

## Transport Protocols

The SDK supports multiple transport protocols for communicating with the Toolbox server. By default, the client uses the latest supported version of the **Model Context Protocol (MCP)**.

You can explicitly select a protocol using the `protocol` option during toolset initialization. This is useful if you need to use the native Toolbox HTTP protocol or pin the client to a specific legacy version of MCP.

{{< notice note>}}
* **Native Toolbox Transport**: This uses the service's native **REST over HTTP** API.
* **MCP Transports**: These options use the **Model Context Protocol over HTTP**.
{{< /notice >}}

### Supported Protocols

| Constant | Description |
| :--- | :--- |
| `Protocol.MCP` | **(Default)** Alias for the default MCP version (currently `2025-06-18`). |
| `Protocol.TOOLBOX` | **DEPRECATED**: The native Toolbox HTTP protocol. Will be removed on March 4, 2026. |
| `Protocol.MCP_v20251125` | MCP Protocol version 2025-11-25. |
| `Protocol.MCP_v20250618` | MCP Protocol version 2025-06-18. |
| `Protocol.MCP_v20250326` | MCP Protocol version 2025-03-26. |
| `Protocol.MCP_v20241105` | MCP Protocol version 2024-11-05. |

{{<  notice note >}}
The **Native Toolbox Protocol** (`Protocol.TOOLBOX`) is deprecated and will be removed on **March 4, 2026**.
Please migrate to using the **MCP Protocol** (`Protocol.MCP`), which is the default.
{{< /notice >}}

### Example

If you wish to use the native Toolbox protocol:

```python
from toolbox_adk import ToolboxToolset
from toolbox_core.protocol import Protocol

toolset = ToolboxToolset(
    server_url="http://127.0.0.1:5000",
    protocol=Protocol.TOOLBOX
)
```

If you want to pin the MCP Version 2025-03-26:

```python
from toolbox_adk import ToolboxToolset
from toolbox_core.protocol import Protocol

toolset = ToolboxToolset(
    server_url="http://127.0.0.1:5000",
    protocol=Protocol.MCP_v20250326
)
```

{{< notice tip>}}
By default, it uses **Toolbox Identity** (no authentication), which is suitable for local development.

For production environments (Cloud Run, GKE) or accessing protected resources, see the [Authentication](#authentication) section for strategies like Workload Identity or OAuth2.
{{< /notice >}}

## Authentication

The `ToolboxToolset` requires credentials to authenticate with the Toolbox server. You can configure these credentials using the `CredentialStrategy` factory methods.

The strategies handle two main types of authentication:
*   **Client-to-Server**: Securing the connection to the Toolbox server (e.g., Workload Identity, API keys).
*   **User Identity**: Authenticating the end-user for specific tools (e.g., 3-legged OAuth).

### 1. Workload Identity (ADC)
*Recommended for Cloud Run, GKE, or local development with `gcloud auth login`.*

Uses the agent's Application Default Credentials (ADC) to generate an OIDC token. This is the standard way for one service to authenticate to another on Google Cloud.

```python
from toolbox_adk import CredentialStrategy, ToolboxToolset

# target_audience: The URL of your Toolbox server
creds = CredentialStrategy.workload_identity(target_audience="https://my-toolbox-service.run.app")

toolset = ToolboxToolset(
    server_url="https://my-toolbox-service.run.app",
    credentials=creds
)
```

### 2. User Identity (OAuth2)
*Recommended for tools that act on behalf of the user.*

Configures the ADK-native interactive 3-legged OAuth flow to get consent and credentials from the end-user at runtime. This strategy is passed to the `ToolboxToolset` just like any other credential strategy.

```python
from toolbox_adk import CredentialStrategy, ToolboxToolset

creds = CredentialStrategy.user_identity(
    client_id="YOUR_CLIENT_ID",
    client_secret="YOUR_CLIENT_SECRET",
    scopes=["https://www.googleapis.com/auth/cloud-platform"]
)

# The toolset will now initiate OAuth flows when required by tools
toolset = ToolboxToolset(
    server_url="...",
    credentials=creds
)
```

### 3. API Key
*Use a static API key passed in a specific header (default: `X-API-Key`).*

```python
from toolbox_adk import CredentialStrategy

# Default header: X-API-Key
creds = CredentialStrategy.api_key(key="my-secret-key")

# Custom header
creds = CredentialStrategy.api_key(key="my-secret-key", header_name="X-My-Header")
```

### 4. HTTP Bearer Token
*Manually supply a static bearer token.*

```python
from toolbox_adk import CredentialStrategy

creds = CredentialStrategy.manual_token(token="your-static-bearer-token")
```

### 5. Manual Google Credentials
*Use an existing `google.auth.credentials.Credentials` object.*

```python
from toolbox_adk import CredentialStrategy
import google.auth

creds_obj, _ = google.auth.default()
creds = CredentialStrategy.manual_credentials(credentials=creds_obj)
```

### 6. Toolbox Identity (No Auth)
*Use this if your Toolbox server does not require authentication (e.g., local development).*

```python
from toolbox_adk import CredentialStrategy

creds = CredentialStrategy.toolbox_identity()
```

### 7. Native ADK Integration
*Convert ADK-native `AuthConfig` or `AuthCredential` objects.*

```python
from toolbox_adk import CredentialStrategy

# From AuthConfig
creds = CredentialStrategy.from_adk_auth_config(auth_config)

# From AuthCredential + AuthScheme
creds = CredentialStrategy.from_adk_credentials(auth_credential, scheme)
```

### 8. Tool-Specific Authentication
*Resolve authentication tokens dynamically for specific tools.*

Some tools may define their own authentication requirements (e.g., Salesforce OAuth, GitHub PAT) via `authSources` in their schema. You can provide a mapping of getters to resolve these tokens at runtime.

```python
async def get_salesforce_token():
    # Fetch token from secret manager or reliable source
    return "sf-access-token"

toolset = ToolboxToolset(
    server_url="...",
    auth_token_getters={
        "salesforce-auth": get_salesforce_token,   # Async callable
        "github-pat": lambda: "my-pat-token"       # Sync callable or static lambda
    }
)
```

## Advanced Configuration

### Additional Headers

You can inject custom headers into every request made to the Toolbox server. This is useful for passing tracing IDs, API keys, or other metadata.

```python
toolset = ToolboxToolset(
    server_url="...",
    additional_headers={
        "X-Trace-ID": "12345",
        "X-My-Header": lambda: get_dynamic_header_value() # Can be a callable
    }
)
```

### Parameter Binding

Bind values to tool parameters globally across all loaded tools. These values will be **fixed** and **hidden** from the LLM.

*   **Schema Hiding**: The bound parameters are removed from the tool schema sent to the model, simplifying the context window.
*   **Auto-Injection**: The values are automatically injected into the tool arguments during execution.

```python
toolset = ToolboxToolset(
    server_url="...",
    bound_params={
        # 'region' will be hidden from the LLM and injected automatically
        "region": "us-central1",
        "api_key": lambda: get_api_key() # Can be a callable
    }
)
```
