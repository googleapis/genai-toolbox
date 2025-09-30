---
title: AlloyDB Admin
linkTitle: AlloyDB Admin
type: docs
weight: 2
description: "The \"alloydb-admin\" source provides a client for the AlloyDB API.\n"
aliases: [/resources/sources/alloydb-admin]
---

## About

The `alloydb-admin` source provides a client to interact with the [Google
AlloyDB API](https://cloud.google.com/alloydb/docs/reference/rest). This allows
tools to perform administrative tasks on AlloyDB resources, such as managing
clusters, instances, and users.

Authentication can be handled in two ways:

1.  **Application Default Credentials (ADC):** By default, the source uses ADC
    to authenticate with the API.
2.  **Client-side OAuth:** If `useClientOAuth` is set to `true`, the source will
    expect an OAuth 2.0 access token to be provided by the client (e.g., a web
    browser) for each request.

## Example

```yaml
sources:
    my-alloydb-admin:
        kind: alloy-admin

    my-oauth-alloydb-admin:
        kind: alloydb-admin
        useClientOAuth: true
```

## Reference

| **field**      | **type** | **required** | **description**                                                                                                                                |
| -------------- | :------: | :----------: | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| kind           |  string  |     true     | Must be "alloydb-admin".                                                                                                                       |
| defaultProject |  string  |     false    | The environment variable containing the Google Cloud project ID to use for all AlloyDB operations.                                             |
| useClientOAuth |  boolean |     false    | If true, the source will use client-side OAuth for authorization. Otherwise, it will use Application Default Credentials. Defaults to `false`. |
