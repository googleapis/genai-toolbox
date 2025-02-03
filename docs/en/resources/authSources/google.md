---
title: "Google Sign-In"
type: docs
weight: 1
description: >
  Use Google Sign-In for Oauth 2.0 flow and token lifecycle. 
---

## Getting Started

Google Sign-In manages the OAuth 2.0 flow and token lifecycle. A user always has
the option to revoke access to an application at any time. To integrate the
Google Sign-In workflow to your web app [follow this
guide](https://developers.google.com/identity/sign-in/web/sign-in).

After setting up the Google Sign-In workflow, you should have registered your
application and retrieved a [Client
ID](https://developers.google.com/identity/sign-in/web/sign-in#create_authorization_credentials).
Configure your auth source in with the `Client ID`.

## Example

```yaml
authSources:
  my-google-auth:
    kind: google
    clientId: YOUR_GOOGLE_CLIENT_ID
```

## Reference

| **field** | **type** | **required** | **description**                                                  |
|-----------|:--------:|:------------:|------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "google".                                                |
| clientId  |  string  |     true     | Client ID of your application from registering your application. |
