# Google OAuth 2.0

To use Google as your Toolbox authentication provider, you could integrate Google sign-in into your application by following this [guide](https://developers.google.com/identity/sign-in/web/sign-in). After seting up the Google sign-in workflow, you should have registered your application and got a [Client ID](https://developers.google.com/identity/sign-in/web/sign-in#create_authorization_credentials). Configure your auth source in `tools.yaml` with this `Client ID`:

```yaml
authSources:
  my-google-auth:       # Your auth source name
    kind: google        # Specify `google` as the auth source kind
    clientId: YOUR_GOOGLE_CLIENT_ID     # Your app's Client ID
```