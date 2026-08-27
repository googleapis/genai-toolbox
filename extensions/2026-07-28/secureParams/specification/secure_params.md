# Secure Parameters Specification

**Extension Identifier:** `com.google.cloud/toolbox.v1`  
**Protocol Version:** `2026-07-28`  
**Schema Definition:** [`../schema/schema.ts`](../schema/schema.ts)

---

## 1. Overview & Motivation

In agentic AI systems, tools frequently require sensitive runtime parameters (such as an end-user `customer_id`, tenant identifier, session token, or user account ID) to isolate data and enforce security boundaries.

Allowing the Large Language Model (LLM) to supply or view these sensitive parameters introduces critical security vulnerabilities:
- **Prompt Injection & Overriding:** An adversary could manipulate the model via prompt injection to alter a `customer_id` and access other tenants' data.
- **Context Window Leakage:** Sensitive user identifiers and tokens are sent in plaintext to third-party LLM providers, appearing in context windows and prompt completion logs.

**Secure Parameters** address this by separating standard tool parameters from secure parameters at the protocol level:
- **Standard Parameters (`inputSchema` / `arguments`):** Advertised to the LLM agent and populated by the model during tool invocation.
- **Secure Parameters (`secureInputSchema` / `secureArguments`):** Hidden from the LLM agent and passed out-of-band directly by the host application.

---

## 2. Capability Negotiation

Clients indicate support for the `com.google.cloud/toolbox.v1` extension by advertising the extension capability within `_meta.clientCapabilities.extensions` during session initialization or within request metadata:

```json
{
  "_meta": {
    "io.modelcontextprotocol/protocolVersion": "2026-07-28",
    "io.modelcontextprotocol/clientInfo": {
      "name": "MyApplicationClient",
      "version": "1.0.0"
    },
    "io.modelcontextprotocol/clientCapabilities": {
      "extensions": {
        "com.google.cloud/toolbox.v1": {}
      }
    }
  }
}
```

---

## 3. Protocol Methods & Behavior

### 3.1 Tool Discovery (`tools/list`)

When a client calls `tools/list`:

- **When Client Supports `com.google.cloud/toolbox.v1`:**
  - Tools defining secure parameters are included in the returned `tools` array.
  - Non-secure parameters are placed in standard `inputSchema`.
  - Secure parameters are separated and placed in `secureInputSchema`.
  - Parameters bound via URL parameters are omitted from both schemas.
- **When Client Does NOT Support `com.google.cloud/toolbox.v1`:**
  - Tools with secure parameters are **excluded (filtered out)** from the `tools` list to prevent unsupported invocations.

#### Example `tools/list` Response

```json
{
  "jsonrpc": "2.0",
  "id": "list-1",
  "result": {
    "tools": [
      {
        "name": "search_customer_records",
        "description": "Searches customer records by query filter.",
        "inputSchema": {
          "type": "object",
          "properties": {
            "query": {
              "type": "string",
              "description": "The search query string."
            }
          },
          "required": ["query"]
        },
        "secureInputSchema": {
          "type": "object",
          "properties": {
            "customer_id": {
              "type": "string",
              "description": "Sensitive customer identifier supplied out-of-band by the calling application."
            }
          },
          "required": ["customer_id"]
        }
      }
    ]
  }
}
```

---

### 3.2 Tool Execution (`tools/call`)

When executing a tool that defines secure parameters:

1. **Extension Check:** If the tool defines secure parameters and the client did not declare `com.google.cloud/toolbox.v1` in `_meta.clientCapabilities.extensions`, execution is rejected with JSON-RPC error `-32002` (`MISSING_REQUIRED_CLIENT_CAPABILITY`).
2. **Argument Separation Validation:**
   - Secure parameters MUST NOT be passed in `arguments`.
   - Non-secure parameters MUST NOT be passed in `secureArguments`.
   - Violations return JSON-RPC error `-32602` (`INVALID_PARAMS`).
3. **Execution:** The server validates and merges `arguments` and `secureArguments` internally before invoking the underlying data source tool.

#### Example `tools/call` Request

```json
{
  "jsonrpc": "2.0",
  "id": "call-1",
  "method": "tools/call",
  "params": {
    "name": "search_customer_records",
    "arguments": {
      "query": "recent transactions"
    },
    "secureArguments": {
      "customer_id": "cust_12345"
    },
    "_meta": {
      "io.modelcontextprotocol/protocolVersion": "2026-07-28",
      "io.modelcontextprotocol/clientCapabilities": {
        "extensions": {
          "com.google.cloud/toolbox.v1": {}
        }
      }
    }
  }
}
```

---

## 4. Error Handling Matrix

| Error Code | Error Constant | Condition | Error Message |
|---|---|---|---|
| `-32002` | `MISSING_REQUIRED_CLIENT_CAPABILITY` | Calling a tool with secure parameters without declaring `com.google.cloud/toolbox.v1` support. | `missing required client capability: tool "<name>" requires com.google.cloud/toolbox.v1 extension which is not supported by the client` |
| `-32602` | `INVALID_PARAMS` | A secure parameter is passed in standard `arguments`. | `parameter "<param_name>" is secure and must not be passed in standard arguments` |
| `-32602` | `INVALID_PARAMS` | A non-secure parameter is passed in `secureArguments`. | `parameter "<param_name>" is not secure and must not be passed in secureArguments` |
| `IsError: true` | Tool Execution Error | A required secure parameter was not provided. | `provided parameters were invalid: parameter "<param_name>" is required` |
