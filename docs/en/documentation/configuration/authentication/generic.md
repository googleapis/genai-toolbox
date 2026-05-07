---
title: "Generic OIDC Auth"
type: docs
weight: 2
description: >
  Use a Generic OpenID Connect (OIDC) provider for OAuth 2.0 flow and token
  lifecycle.
---

## Getting Started

The Generic Auth Service allows you to integrate with any OpenID Connect (OIDC)
compliant identity provider (IDP). It discovers the JWKS (JSON Web Key Set) URL
either through the provider's `/.well-known/openid-configuration` endpoint or
directly via the provided `authorizationServer`.

To configure this auth service, you need to provide the `audience` (the expected `aud` claim in the token), the `authorizationServer` of your identity provider, and optionally a list of `scopesRequired` that must be present in the token's claims.

> [!NOTE]
> The only time the `aud` claim matches the `client_id` is inside an ID Token (a concept from OpenID Connect used to verify a user's identity). Because an ID token is intended to be consumed by the client application itself, the client is the audience.

## Usage Modes

The Generic Auth Service supports two distinct modes of operation:

### 1. Toolbox Auth

This mode is used for Toolbox's native authentication/authorization features. It
is active when you reference the auth service in a tool's configuration and
`mcpEnabled` is set to false.

- **Header**: Expects the token in a custom header matching `<name>_token`
  (e.g., `my-generic-auth_token`).
- **Token Type**: Only supports **JWT** (OIDC) tokens.
- **Usage**: Used for [Authenticated Parameters][auth-params] and [Authorized
  Invocations][auth-invoke].

#### Token Validation

When a request is received in this mode, the service will:

1. Extract the token from the `<name>_token` header.
2. Treat it as a JWT (opaque tokens are not supported in this mode).
3. Validates signature using JWKS fetched from `authorizationServer`.
4. Verifies expiration (`exp`) and audience (`aud`).
5. Verifies required scopes in `scope` claim.

#### Example

```yaml
kind: authServices
name: my-generic-auth
type: generic
audience: ${YOUR_OIDC_AUDIENCE}
authorizationServer: https://your-idp.example.com
# mcpEnabled: false
scopesRequired:
  - read
  - write
```

#### Tool Usage Example

To use this auth service for **Authenticated Parameters** or **Authorized
Invocations**, reference it in your tool configuration:

```yaml
kind: tool
name: secure_query
type: postgres-sql
source: my-pg-instance
statement: |
  SELECT * FROM data WHERE user_id = $1
parameters:
  - name: user_id
    type: strings
    description: Auto-populated from token
    authServices:
      - name: my-generic-auth
        field: sub # Extract 'sub' claim from JWT
authRequired:
  - my-generic-auth # Require valid token for invocation
```

### 2. MCP Authorization

This mode enforces global authentication for all MCP endpoints. It is active
when `mcpEnabled` is set to `true` in the auth service configuration.

- **Header**: Expects the token in the standard `Authorization: Bearer <token>`
  header.
- **Token Type**: Supports both **JWT** and **Opaque** tokens.
- **Usage**: Used to secure the entire MCP server.

#### Token Validation

When a request is received in this mode, the service will:

1. Extract the token from the `Authorization` header after `Bearer ` prefix.
2. Determine if the token is a JWT or an opaque token based on format (JWTs
   contain exactly two dots).
3. For **JWTs**:
   - Validates signature using JWKS fetched from `authorizationServer`.
   - Verifies expiration (`exp`) and audience (`aud`).
   - Verifies required scopes in `scope` claim.
4. For **Opaque Tokens**:
   - Calls the introspection endpoint (either configured via `introspectionEndpoint`
     or discovered from the `authorizationServer`'s OIDC configuration).
   - Verifies expiration (`exp`) and audience (`aud` or `"audience"` fallback).
   - Verifies required scopes in `scope` field.

#### Example

```yaml
kind: authServices
name: my-generic-auth
type: generic
audience: ${YOUR_TOKEN_AUDIENCE}
authorizationServer: https://your-idp.example.com
mcpEnabled: true
scopesRequired:
  - read
  - write
```

#### Google Opaque Access Token Validation Example

To use Google's `tokeninfo` endpoint for validating opaque access tokens, configure the service to use the `GET` method and `access_token` parameter name:

```yaml
kind: authServices
name: google-auth
type: generic
audience: "YOUR_GOOGLE_CLIENT_ID.apps.googleusercontent.com"
authorizationServer: https://accounts.google.com
introspectionEndpoint: https://www.googleapis.com/oauth2/v3/tokeninfo
introspectionMethod: GET
introspectionParamName: access_token
mcpEnabled: true
```

#### Okta OIDC Configuration Example

To secure your MCP server or tools using Okta as the identity provider:

```yaml
kind: authServices
name: okta-auth
type: generic
audience: api://default # Or your custom Okta audience
authorizationServer: https://your-subdomain.okta.com/oauth2/default
mcpEnabled: true
scopesRequired:
  - openid
  - profile
```

> [!NOTE]
> If you are using Okta's Org Authorization Server (instead of a Custom Authorization Server), your `authorizationServer` URL will be `https://your-subdomain.okta.com`.

{{< notice tip >}} Use environment variable replacement with the format
${ENV_NAME} instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

[auth-invoke]: ../tools/_index.md#authorized-invocations
[auth-params]: ../tools/_index.md#authenticated-parameters
[mcp-auth]: https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization

## Reference

| **field**              | **type** | **required** | **description**                                                                                                                                                                                       |
| ---------------------- | :------: | :----------: | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| type                   |  string  |     true     | Must be "generic".                                                                                                                                                                                    |
| audience               |  string  |     true     | The expected audience (`aud` claim) in the token. This ensures the token was minted specifically for your application. See [Getting Started](#getting-started) for details on OIDC audience matching. |
| authorizationServer    |  string  |     true     | The base URL of your OIDC provider. The service will append `/.well-known/openid-configuration` to discover the JWKS URI. HTTP is allowed but logs a warning.                                         |
| mcpEnabled             |   bool   |    false     | Indicates if MCP endpoint authentication should be applied. Defaults to false.                                                                                                                        |
| scopesRequired         | []string |    false     | A list of required scopes that must be present in the token's `scope` claim to be considered valid.                                                                                                   |
| introspectionEndpoint  |  string  |    false     | Optional override for the token introspection URL. Useful if the provider does not list it in OIDC discovery (e.g., Google).                                                                          |
| introspectionMethod    |  string  |    false     | HTTP method to use for introspection. Defaults to "POST". Set to "GET" for providers like Google.                                                                                                     |
| introspectionParamName |  string  |    false     | Parameter name for the token in the introspection request. Defaults to "token". Set to "access_token" for Google.                                                                                     |
