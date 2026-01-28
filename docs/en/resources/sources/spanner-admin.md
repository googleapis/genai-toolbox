---
title: Spanner Admin
type: docs
weight: 1
description: "A \"spanner-admin\" source provides a client for the Cloud Spanner Admin API.\n"
alias: [/resources/sources/spanner-admin]
---

## About

The `spanner-admin` source provides a client to interact with the [Google
Cloud Spanner Admin API](https://cloud.google.com/spanner/docs/reference/rpc/google.spanner.admin.instance.v1). This
allows tools to perform administrative tasks on Spanner instances, such as
creating instances.

Authentication can be handled in two ways:

1.  **Application Default Credentials (ADC):** By default, the source uses ADC
    to authenticate with the API.
2.  **Client-side OAuth:** If `useClientOAuth` is set to `true`, the source will
    expect an OAuth 2.0 access token to be provided by the client (e.g., a web
    browser) for each request.

## Example

```yaml
kind: sources
name: my-spanner-admin
type: spanner-admin
---
kind: sources
name: my-oauth-spanner-admin
type: spanner-admin
useClientOAuth: true
```

## Reference

| **field**      | **type** | **required** | **description**                                                                                                                                |
| -------------- | :------: | :----------: | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| type           |  string  |     true     | Must be "spanner-admin".                                                                                                                       |
| defaultProject |  string  |     false    | The Google Cloud project ID to use for Spanner infrastructure tools.                                                                           |
| useClientOAuth |  boolean |     false    | If true, the source will use client-side OAuth for authorization. Otherwise, it will use Application Default Credentials. Defaults to `false`. |
