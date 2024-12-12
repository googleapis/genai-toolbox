# Auth Sources

Auth Sources represent authentication sources that a tool can interact with. Toolbox supports authentication providers that conform to the [OpenID Connect (OIDC) protocol](https://openid.net/developers/how-connect-works/). You can define Auth Sources as a map in the `authSources` section of your `tools.yaml` file. Typically, an Auth Source configuration will contain information required to verify token authenticity. For OIDC tokens, you should provide the `kind` of the auth provider and your `clientId` for it to be verified.

```yaml
authSources:
  my-google-auth:
    kind: google
    clienId: YOUR_GOOGLE_CLIENT_ID
```

If you are accessing Toolbox with multiple applications, each application should register their own Client ID even if they use the same `kind` of auth provider.

## Kinds of Auth Sources

We currently support the following types of kinds of auth sources:

* [Google OAuth 2.0](./google.md) - Authenticate with a Google-signed OpenID Connect (OIDC) ID token.
