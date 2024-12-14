# Auth Sources

Auth Sources represent authentication sources that a tool can interact with. Toolbox supports authentication providers that conform to the [OpenID Connect (OIDC) protocol](https://openid.net/developers/how-connect-works/). You can define Auth Sources as a map in the `authSources` section of your `tools.yaml` file. Typically, an Auth Source configuration will contain information required to verify token authenticity. You should provide the `kind` of the auth provider and your `clientId` for your registered app to allow Toolbox verify the authentication token.

## Example

```yaml
authSources:
  my-google-auth:
    kind: google
    clientId: YOUR_GOOGLE_CLIENT_ID
```

Tip: If you are accessing Toolbox with multiple applications, each application should register their own Client ID even if they use the same `kind` of auth provider.

## Kinds of Auth Sources

We currently support the following types of kinds of auth sources:

* [Google OAuth 2.0](./google.md) - Authenticate with a Google-signed OpenID Connect (OIDC) ID token.

## ID Token

The OIDC authentication workflow transmit user information with ID tokens. ID tokens are JSON Web Tokens (JWTs) that are composed of a set of key-value pairs called [claims](https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims). ID tokens can include claims such as user ID, user name, user emails etc. After specifying `authSources`, you can configure your tool's authenticated parameters by following this [guide](../tools/README.md#authenticated-parameters)
