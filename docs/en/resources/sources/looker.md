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

### Database User

This source only uses API authentication. You will need to
[create an API user][looker-user] to login to Looker.

[looker-user]:
    https://cloud.google.com/looker/docs/api-auth#authentication_with_an_sdk

## Example

```yaml
sources:
    my-looker-source:
        kind: looker
        base_url: http://looker.example.com
        client_id: ${LOOKER_CLIENT_ID}
        client_secret: ${LOOKER_CLIENT_SECRET}
        verify_ssl: true
        timeout: 600s
```

The Looker base url will look like "https://looker.example.com", don't include
a trailing "/". In some cases, especially if your Looker is deployed
on-premises, you may need to add the API port numner like
"https://looker.example.com:19999".

Verify ssl should almost always be "true" (all lower case) unless you are using
a self-signed ssl certificate for the Looker server. Anything other than "true"
will be interpretted as false.

The client id and client secret are seemingly random character sequences
assigned by the looker server.

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field**     | **type** | **required** | **description**                                                                           |
| ------------- | :------: | :----------: | ----------------------------------------------------------------------------------------- |
| kind          |  string  |     true     | Must be "looker".                                                                         |
| base_url      |  string  |     true     | The URL of your Looker server with no trailing /).                                        |
| client_id     |  string  |     true     | The client id assigned by Looker.                                                         |
| client_secret |  string  |     true     | The client secret assigned by Looker.                                                     |
| verify_ssl    |  string  |     true     | Whether to check the ssl certificate of the server.                                       |
| timeout       |  string  |    false     | Maximum time to wait for query execution (e.g. "30s", "2m"). By default, 120s is applied. |
