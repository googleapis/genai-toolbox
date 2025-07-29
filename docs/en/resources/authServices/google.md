---
title: "Google Sign-In"
type: docs
weight: 1
description: >
  Use Google Sign-In for Oauth 2.0 flow and token lifecycle. 
---

## Getting Started

Google Sign-In manages the OAuth 2.0 flow and token lifecycle. To integrate the
Google Sign-In workflow to your web app [follow this guide][gsi-setup].

After setting up the Google Sign-In workflow, you should have registered your
application and retrieved a [Client ID][client-id]. Configure your auth service
in with the `Client ID`.

[gsi-setup]: https://developers.google.com/identity/sign-in/web/sign-in
[client-id]: https://developers.google.com/identity/sign-in/web/sign-in#create_authorization_credentials

## Behavior

### Authorized Invocations

When using [Authorized Invocations][auth-invoke], a tool will be
considered authorized if it has a valid Oauth 2.0 token that matches the Client
ID.

[auth-invoke]: ../tools/#authorized-invocations

### Authenticated Parameters

When using [Authenticated Parameters][auth-params], any [claim provided by the
id-token][provided-claims] can be used for the parameter.

[auth-params]: ../tools/#authenticated-phugarameters
[provided-claims]:
    https://developers.google.com/identity/openid-connect/openid-connect#obtaininguserprofileinformation

## Example

```yaml
authServices:
  my-google-auth:
    kind: google
    clientId: ${YOUR_GOOGLE_CLIENT_ID}
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field** | **type** | **required** | **description**                                                  |
|-----------|:--------:|:------------:|------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "google".                                                |
| clientId  |  string  |     true     | Client ID of your application from registering your application. |
