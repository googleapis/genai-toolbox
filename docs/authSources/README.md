# AuthSources

`AuthSources` represent authentication sources that a tool can interact with.
Toolbox supports authentication providers that conform to the [OpenID Connect
(OIDC) protocol](https://openid.net/developers/how-connect-works/). You can
define Auth Sources as a map in the `authSources` section of your `tools.yaml`
file. Typically, an Auth Source is required for the following features:

- [Authenticated parameters](../tools/README.md#authenticated-parameters)
- [Authorized tool call](../tools/README.md#authorized-tool-call)

## Example

```yaml
authSources:
  my-google-auth:
    kind: google
    clientId: YOUR_GOOGLE_CLIENT_ID
```

> [!TIP]
> If you are accessing Toolbox with multiple applications, each application
> should register their own Client ID even if they use the same `kind` of auth
> provider.
>
> Here's an example:
>
> ```yaml
> authSources:
>     my_auth_app_1:
>         kind: google
>         client_id: YOUR_CLIENT_ID_1
>     my_auth_app_2:
>         kind: google
>         client_id: YOUR_CLIENT_ID_2
>
> tools:
>     my_tool:
>         parameters:
>             - name: user_id
>               type: string
>               auth_sources:
>                   - name: my_auth_app_1
>                     field: sub
>                   - name: my_auth_app_2
>                     field: sub
>         ...
>
>     my_tool_no_param:
>         auth_required:
>             - my_auth_app_1
>             - my_auth_app_2
>         ...
> ```

## Kinds of authSources

We currently support the following types of kinds of `authSources`:

- [Google OAuth 2.0](./google.md) - Authenticate with a Google-signed OpenID
  Connect (OIDC) ID token.

## ID Token

The OIDC authentication workflow transmit user information with ID tokens. ID
tokens are JSON Web Tokens (JWTs) that are composed of a set of key-value pairs
called
[claims](https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims).
ID tokens can include claims such as user ID, user name, user emails etc. After
specifying `authSources`, you can configure your tool's authenticated parameters
by following this [guide](../tools/README.md#authenticated-parameters)

## Usage

`AuthSources` can be used for both `authorization` and `authentication`:

- `Authorization` verifies that a Tool invocation request includes the necessary
  authentication token. Add an authorization layer to your Tool calling by
  configuring the [authorized Tool
  call](../tools/README.md#authorized-tool-call).
- `Authentication` verifies the user's identity in a Tool's query to the
  database. Configure [authenticated
  parameters](../tools/README.md#authenticated-parameters) to auto-populate your
  Tool parameters from user login info.

After confuring your `authSources`, use Toolbox Client SDK to add your `ID tokens` to
the header of a Tool invocation request.
