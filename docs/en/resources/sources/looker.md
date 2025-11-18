---
title: "Looker"
type: docs
weight: 1
description: >
  Looker is a business intelligence tool that also provides a semantic layer.
---

## About

[Looker][looker-docs] is a web based business intelligence and data management
tool that provides a semantic layer to facilitate querying. It can be deployed
in the cloud, on GCP, or on premises.

[looker-docs]: https://cloud.google.com/looker/docs

## Requirements

### Looker User

This source only uses API authentication. You will need to
[create an API user][looker-user] to login to Looker.

[looker-user]:
    https://cloud.google.com/looker/docs/api-auth#authentication_with_an_sdk

{{< notice note >}}
To use the Conversational Analytics API, you will need to have the following
Google Cloud Project API enabled and IAM permissions.
{{< /notice >}}

### API Enablement in GCP

Enable the following APIs in your Google Cloud Project:

```
gcloud services enable geminidataanalytics.googleapis.com --project=$PROJECT_ID
gcloud services enable cloudaicompanion.googleapis.com --project=$PROJECT_ID
```

### IAM Permissions in GCP

In addition to [setting the ADC for your server][set-adc], you need to ensure
the IAM identity has been given the following IAM roles (or corresponding
permissions):

- `roles/looker.instanceUser`
- `roles/cloudaicompanion.user`
- `roles/geminidataanalytics.dataAgentStatelessUser`

To initialize the application default credential run `gcloud auth login
--update-adc` in your environment before starting MCP Toolbox.

[set-adc]: https://cloud.google.com/docs/authentication/provide-credentials-adc

## Example

```yaml
sources:
    my-looker-source:
        kind: looker
        base_url: http://looker.example.com
        client_id: ${LOOKER_CLIENT_ID}
        client_secret: ${LOOKER_CLIENT_SECRET}
        project: ${LOOKER_PROJECT}
        location: ${LOOKER_LOCATION}
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
these settings

The `project` and `location` fields are utilized **only** when using the
conversational analytics tool.

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field**            | **type** | **required** | **description**                                                                           |
| -------------------- | :------: | :----------: | ----------------------------------------------------------------------------------------- |
| kind                 |  string  |     true     | Must be "looker".                                                                         |
| base_url             |  string  |     true     | The URL of your Looker server with no trailing /).                                        |
| client_id            |  string  |    false     | The client id assigned by Looker.                                                         |
| client_secret        |  string  |    false     | The client secret assigned by Looker.                                                     |
| verify_ssl           |  string  |    false     | Whether to check the ssl certificate of the server.                                       |
| project              |  string  |    false     | The project id to use in Google Cloud.                                                    |
| location             |  string  |    false     | The location to use in Google Cloud. (default: us)                                        |
| timeout              |  string  |    false     | Maximum time to wait for query execution (e.g. "30s", "2m"). By default, 120s is applied. |
| use_client_oauth     |  string  |    false     | Use OAuth tokens instead of client_id and client_secret. (default: false)                 |
| show_hidden_models   |  string  |    false     | Show or hide hidden models. (default: true)                                               |
| show_hidden_explores |  string  |    false     | Show or hide hidden explores. (default: true)                                             |
| show_hidden_fields   |  string  |    false     | Show or hide hidden fields. (default: true)                                               |
