---
title: "Cloud Logging Admin Source"
type: docs
linkTitle: "Source"
weight: 1
description: >
  The Cloud Logging Admin source enables tools to interact with the Cloud Logging API, allowing for the retrieval of log names, monitored resource types, and the querying of log data.
no_list: true
---

## About

The Cloud Logging Admin source provides a client to interact with the [Google
Cloud Logging API](https://cloud.google.com/logging/docs). This allows tools to list log names, monitored resource types, and query log entries.

Authentication can be handled in two ways:

1.  **Application Default Credentials (ADC):** By default, the source uses ADC
    to authenticate with the API.
2.  **Client-side OAuth:** If `useClientOAuth` is set to `true`, the source will
    expect an OAuth 2.0 access token to be provided by the client (e.g., a web
    browser) for each request.

## Available Tools

{{< list-tools >}}

## Example

Initialize a Cloud Logging Admin source that uses ADC:

```yaml
kind: source
name: my-cloud-logging
type: cloud-logging-admin
project: my-project-id
```

Initialize a Cloud Logging Admin source that uses client-side OAuth:

```yaml
kind: source
name: my-oauth-cloud-logging
type: cloud-logging-admin
project: my-project-id
useClientOAuth: true
```

Initialize a Cloud Logging Admin source that uses service account impersonation:

```yaml
kind: source
name: my-impersonated-cloud-logging
type: cloud-logging-admin
project: my-project-id
impersonateServiceAccount: "my-service-account@my-project.iam.gserviceaccount.com"
```

## Reference

| **field**                   | **type** | **required** | **description**                                                                                                                                                                                 |
|-----------------------------|:--------:|:------------:|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| type                        |  string  |     true     | Must be "cloud-logging-admin".                                                                                                                                                                  |
| project                     |  string  |     true     | ID of the GCP project.                                                                                                                                                                          |
| useClientOAuth              | boolean  |    false     | If true, the source will use client-side OAuth for authorization. Otherwise, it will use Application Default Credentials. Defaults to `false`. Cannot be used with `impersonateServiceAccount`. |
| impersonateServiceAccount   |  string  |    false     | The service account to impersonate for API calls. Cannot be used with `useClientOAuth`.                                                                                                         |
