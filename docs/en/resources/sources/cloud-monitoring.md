---
title: "Cloud Monitoring"
type: docs
weight: 1
description: >
  A "cloud-monitoring" source provides a client for the Cloud Monitoring API.
aliases:
- /resources/sources/cloud-monitoring
---

## About

The `cloud-monitoring` source provides a client to interact with the [Google
Cloud Monitoring API](https://cloud.google.com/monitoring/api). This allows
tools to access cloud monitoring metrics explorer and run promql queries.

Authentication can be handled in two ways:
1.  **Application Default Credentials (ADC):** By default, the source uses ADC
    to authenticate with the API.
2.  **Client-side OAuth:** If `useClientOAuth` is set to `true`, the source will
    expect an OAuth 2.0 access token to be provided by the client (e.g., a web
    browser) for each request.

## Example

```yaml
sources:
    my-cloud-monitoring:
        kind: cloud-monitoring

    my-oauth-cloud-monitoring:
        kind: cloud-monitoring
        useClientOAuth: true
```

## Reference

| **field**      | **type** | **required** | **description**                                                                                                                                |
|----------------|:--------:|:------------:|------------------------------------------------------------------------------------------------------------------------------------------------|
| kind           |  string  |     true     | Must be "cloud-monitoring".                                                                                                                    |
| useClientOAuth | boolean  |    false     | If true, the source will use client-side OAuth for authorization. Otherwise, it will use Application Default Credentials. Defaults to `false`. |
