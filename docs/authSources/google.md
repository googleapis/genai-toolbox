# Google OAuth 2.0

To use Google as your Toolbox authentication provider, you could integrate Google sign-in into your application by following this [guide](https://developers.google.com/identity/sign-in/web/sign-in). After setting up the Google sign-in workflow, you should have registered your application and retrieved a [Client ID](https://developers.google.com/identity/sign-in/web/sign-in#create_authorization_credentials). Configure your auth source in `tools.yaml` with the `Client ID`.

## Example

```yaml
authSources:
  my-google-auth:
    kind: google
    clientId: YOUR_GOOGLE_CLIENT_ID
```

## Reference

| **field** | **type** | **required** | **description**                                                              |
|-----------|:--------:|:------------:|------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "google".                                                  |
| clientId  |  string  |     true     | Client ID of your application from registering your app on GCP.    |

## Usage

An auth source can be used for both `authorization` and `authentication`:

- `Authorization` verifies that a Tool invocation request includes the necessary authentication token.
- `Authentication` verifies the user's identity in a Tool's query to the database.


