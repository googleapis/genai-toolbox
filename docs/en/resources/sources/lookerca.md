---
title: "Looker Conversational Analytics"
type: docs
weight: 1
description: >
  Looker is a business intelligence tool that also provides a semantic layer.
  This source is used to access Looker with the Conversational Analytics API.
---

## About

[Looker][looker-docs] is a web based business intelligence and data management
tool that provides a semantic layer to facilitate querying. It can be deployed
in the cloud, on GCP, or on premises.

[looker-docs]: https://cloud.google.com/looker/docs

## Requirements

### Looker User

This source uses API authentication. You will need to
[create an API user][looker-user] to login to Looker.
If you want to use OAuth see the sample setup for
[Looker Gemini CLI OAuth](../../samples/looker/looker_gemini_oauth).

[looker-user]:
    https://cloud.google.com/looker/docs/api-auth#authentication_with_an_sdk

### Google Cloud Project

Enable the following APIs in your Google Cloud Project:

```
gcloud services enable geminidataanalytics.googleapis.com --project=project_id
gcloud services enable cloudaicompanion.googleapis.com --project=project_id
gcloud services enable bigquery.googleapis.com --project=project_id
```

### Google Cloud User

You will need an Application Default Credential for Google Cloud. Your Google Cloud
user must have the role `roles/looker.instanceUser`.

## Example

```yaml
sources:
    my-lookerca-source:
        kind: lookerca
        base_url: http://looker.example.com
        client_id: ${LOOKERCA_CLIENT_ID}
        client_secret: ${LOOKERCA_CLIENT_SECRET}
        use_client_oauth: ${LOOKERCA_USE_CLIENT_OAUTH}
        project: ${LOOKERCA_PROJECT}
        location: ${LOOKERCA_LOCATION}
        verify_ssl: true
        timeout: 600s
```

The Looker base url will look like "https://looker.example.com", don't include
a trailing "/". In some cases, especially if your Looker is deployed
on-premises, you may need to add the API port number like
"https://looker.example.com:19999".

Verify ssl should almost always be "true" (all lower case) unless you are using
a self-signed ssl certificate for the Looker server. Anything other than "true"
will be interpreted as false.

The client id and client secret are seemingly random character sequences
assigned by the looker server. If you are using Looker OAuth you don't need
these settings.

The project value will be the project id of the associated Google Cloud project.

The location will be the location code for the project, e.g. 'us'.

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

In the session where you will start the MCP Toolbox, set the environment
variables and run `gcloud auth login --update-adc` before starting the toolbox.

## Reference

| **field**        | **type** | **required** | **description**                                                                           |
| ---------------- | :------: | :----------: | ----------------------------------------------------------------------------------------- |
| kind             |  string  |     true     | Must be "looker".                                                                         |
| base_url         |  string  |     true     | The URL of your Looker server with no trailing /).                                        |
| client_id        |  string  |    false     | The client id assigned by Looker.                                                         |
| client_secret    |  string  |    false     | The client secret assigned by Looker.                                                     |
| verify_ssl       |  string  |    false     | Whether to check the ssl certificate of the server.                                       |
| timeout          |  string  |    false     | Maximum time to wait for query execution (e.g. "30s", "2m"). By default, 120s is applied. |
| use_client_oauth |  string  |    false     | Use OAuth tokens instead of client_id and client_secret. (default: false)                 |
| project          |  string  |    false     | The project id to use in Google Cloud.                                                    |
| location         |  string  |    false     | The location to use in Google Cloud. (default: us)                                        |
