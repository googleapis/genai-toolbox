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

To configure this auth service, you need to provide the `audience` (typically
your client ID or the intended audience for the token), the
`authorizationServer` of your identity provider, and optionally a list of
`scopesRequired` that must be present in the token's claims.

## Behavior

### Token Validation

When a request is received, the service will:

1. Extract the token from the `<name>_token` header (e.g.,
   `my-generic-auth_token`).
2. Fetch the JWKS from the configured `authorizationServer` (caching it in the
   background) to verify the token's signature.
3. Validate that the token is not expired and its signature is valid.
4. Verify that the `aud` (audience) claim matches the configured `audience`.
   claim contains all required scopes.
5. Return the validated claims to be used for [Authenticated
   Parameters][auth-params] or [Authorized Invocations][auth-invoke].

[auth-invoke]: ../tools/_index.md#authorized-invocations
[auth-params]: ../tools/_index.md#authenticated-parameters

## Example

```yaml
kind: authServices
name: my-generic-auth
type: generic
audience: ${YOUR_OIDC_AUDIENCE}
authorizationServer: https://your-idp.example.com
mcpEnabled: false
scopesRequired:
  - read
  - write
```

{{< notice tip >}} Use environment variable replacement with the format
${ENV_NAME} instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field**           | **type** | **required** | **description**                                                                                                                                               |
| ------------------- | :------: | :----------: | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| type                |  string  |     true     | Must be "generic".                                                                                                                                            |
| audience            |  string  |     true     | The expected audience (`aud` claim) in the JWT token. This ensures the token was minted specifically for your application.                                    |
| authorizationServer |  string  |     true     | The base URL of your OIDC provider. The service will append `/.well-known/openid-configuration` to discover the JWKS URI. HTTP is allowed but logs a warning. |
| mcpEnabled          |   bool   |    false     | Indicates if MCP endpoint authentication should be applied. Defaults to false.                                                                                |
| scopesRequired      | []string |    false     | A list of required scopes that must be present in the token's `scope` claim to be considered valid.                                                           |
