---
title: "Gemini Data Analytics"
type: docs
weight: 1
description: >
  A "cloud-gemini-data-analytics" source provides a client for the Gemini Data Analytics API.
aliases:
- /resources/sources/cloud-gemini-data-analytics
---

## About

The `cloud-gemini-data-analytics` source provides a client to interact with the [Gemini Data Analytics API](https://docs.cloud.google.com/gemini/docs/conversational-analytics-api/reference/rest). This allows tools to send natural language queries to the API.

Authentication can be handled in two ways:

1.  **Application Default Credentials (ADC):** By default, the source uses ADC to authenticate with the API.
2.  **Client-side OAuth:** If `useClientOAuth` is set to `true`, the source will expect an OAuth 2.0 access token to be provided by the client (e.g., a web browser) for each request.

## Example

```yaml
sources:
    my-gda-source:
        kind: cloud-gemini-data-analytics
        projectId: my-project-id

    my-oauth-gda-source:
        kind: cloud-gemini-data-analytics
        projectId: my-project-id
        useClientOAuth: true
```

## Reference

| **field**      | **type** | **required** | **description**                                                                                                                                |
|----------------|:--------:|:------------:|------------------------------------------------------------------------------------------------------------------------------------------------|
| kind           |  string  |     true     | Must be "cloud-gemini-data-analytics".                                                                                                         |
| projectId      |  string  |     true     | The Google Cloud Project ID where the API is enabled.                                                                                          |
| useClientOAuth | boolean  |    false     | If true, the source will use client-side OAuth for authorization. Otherwise, it will use Application Default Credentials. Defaults to `false`. |
