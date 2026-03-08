# Secure Parameters Design Document

## 1. Overview

Secure parameters provide a mechanism for tools to accept sensitive data (like tokens or secrets) that should not be logged, exposed in standard parameter lists, or mixed with normal operational arguments.

This document outlines the design decisions made to integrate secure parameters into the MCP Toolbox while maintaining strict backward compatibility for all existing tools.

## 2. Core Requirements

The design was guided by three primary requirements:

1.  **YAML API Consistency:** Secure parameters must be defined as a property (`secure: true`) on standard parameters within the `parameters` list, rather than requiring a separate list or schema structure.
2.  **Namespace Collisions:** A regular parameter and a secure parameter must be allowed to share the same name (e.g., a standard `id` for query routing and a secure `id` used within a database secure context).
3.  **Zero Interface Changes (Backward Compatibility):** Existing tools must not require *any* code modifications. They must continue to receive standard parameters exactly as before, entirely shielded from the existence of secure parameters.

## 3. Design Decisions

### 3.1 Transport via Protocol `_meta`
To separate secure parameters from standard operational parameters over the wire, we utilize the Model Context Protocol `_meta` extension fields.
*   **Tool List (`tools/list`):** The schema for secure parameters is published under `_meta.toolbox/stateSchema`, rather than mixing it into the standard `inputSchema`.
*   **Tool Execution (`tools/call`):** Secure parameters are passed under `_meta.toolbox/state` rather than within the standard `arguments` map.

### 3.2 Parsing and Data Structures (`ParamValue`)
The most significant challenge was passing these split namespaces into the `Tool.Invoke()` interface without altering its signature (`func Invoke(..., params ParamValues, ...)`).

*   **Annotation:** We extended the `ParamValue` struct with an `IsSecure bool` annotation.
*   **Encapsulation:** The MCP server parses both standard `arguments` and `_meta.toolbox/state` independently, flags them appropriately, and concatenates them into a single `ParamValues` slice. Because they are stored in a slice, name collisions (`id`: standard, `id`: secure) are natively supported.
*   **Shielding:** We modified the standard `params.AsMap()` method to strictly **filter out** any parameter marked `IsSecure`.
    *   *Result:* Because all existing tools call `AsMap()` to retrieve their bindings, they remain entirely unaffected by secure parameters.
*   **Opt-In Access:** We introduced a new `params.AsSecureMap()` method. Tools that explicitly require secure parameters (like Spanner) can call this method to retrieve the separate namespace.

## 4. API Examples

### 4.1 YAML Configuration API
Tools are configured via YAML. To mark a parameter as secure, simply add the `secure: true` property to its definition. 

```yaml
tools:
  my-secure-tool:
    kind: spanner-sql
    source: my-instance
    description: "Tool with colliding namespaces"
    statement: "SELECT * FROM users WHERE id = @id AND token = SECURE_CONTEXT("id")"
    parameters:
      # This is the standard parameter used for @id
      - name: id
        type: integer
        description: "The regular ID"
      # This is the secure parameter used for SECURE_CONTEXT
      - name: id
        type: string
        secure: true
        description: "The secure ID"
```

### 4.2 JSONRPC API
The MCP protocol relies on JSONRPC.

**1. Discovery (`tools/list`)**
The Toolbox exposes the secure parameter requirement in the `_meta` object:

```json
{
  "name": "my-secure-tool",
  "description": "Tool with colliding namespaces",
  "inputSchema": {
    "type": "object",
    "properties": {
      "id": { "type": "integer", "description": "The regular ID" }
    },
    "required": ["id"]
  },
  "_meta": {
    "toolbox/stateSchema": {
      "type": "object",
      "properties": {
        "id": { "type": "string", "description": "The secure ID" }
      },
      "required": ["id"]
    }
  }
}
```

**2. Execution (`tools/call`)**
The caller must provide the secure data in the `_meta.toolbox/state` object:

```json
{
  "method": "tools/call",
  "params": {
    "name": "my-secure-tool",
    "arguments": {
      "id": 12345
    },
    "_meta": {
      "toolbox/state": {
        "id": "super-secret-token-string"
      }
    }
  }
}
```

### 4.3 Tool Implementation API (Go)
When building a new tool or updating an existing one to support secure parameters, the implementation is straightforward. The tool relies on the `ParamValues` slice passed into its `Invoke` method.

```go
func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
    
    // 1. Retrieve standard parameters (e.g., the integer 12345)
    // This is what all existing tools do today. It automatically filters out secure params.
    standardParams := params.AsMap()
    
    // 2. Opt-in to retrieve secure parameters (e.g., "super-secret-token-string")
    secureParams := params.AsSecureMap()

    // ... execute logic using both namespaces ...
    
    return results, nil
}
```
