---
title: "Cloud SQL Admin"
type: docs
weight: 1
description: >
  A "cloud-sql-admin" source provides a client for the Cloud SQL Admin API.
aliases:
- /resources/sources/cloud-sql-admin
---

## About

The `cloud-sql-admin` source provides a client to interact with the [Google
Cloud SQL Admin API](https://cloud.google.com/sql/docs/mysql/admin-api). This
allows tools to perform administrative tasks on Cloud SQL instances, such as
creating users and databases.

Authentication can be handled in two ways:
1.  **Application Default Credentials (ADC):** By default, the source uses ADC
    to authenticate with the API.
2.  **Client-side OAuth:** If `useClientOAuth` is set to `true`, the source will
    expect an OAuth 2.0 access token to be provided by the client (e.g., a web
    browser) for each request.

## Example

```yaml
sources:
    my-cloud-sql-admin:
        kind: cloud-sql-admin

    my-oauth-cloud-sql-admin:
        kind: cloud-sql-admin
        useClientOAuth: true
```

## Reference

| **field**      | **type** | **required** | **description**                                                                                                                                |
|----------------|:--------:|:------------:|------------------------------------------------------------------------------------------------------------------------------------------------|
| kind           |  string  |     true     | Must be "cloud-sql-admin".                                                                                                                     |
| useClientOAuth | boolean  |    false     | If true, the source will use client-side OAuth for authorization. Otherwise, it will use Application Default Credentials. Defaults to `false`. |
