---
title: "HTTP"
linkTitle: "HTTP"
type: docs
weight: 1
description: >
  The HTTP source enables the Toolbox to retrieve data from a remote server using HTTP requests.
---

## About

The HTTP Source allows Toolbox to retrieve data from arbitrary HTTP
endpoints. This enables Generative AI applications to access data from web APIs
and other HTTP-accessible resources.

## Available Tools

- [`http`](../tools/http/http.md)  
  Make HTTP requests to REST APIs or other web services.

## Example

```yaml
sources:
  my-http-source:
    kind: http
    baseUrl: https://api.example.com/data
    timeout: 10s # default to 30s
    headers:
      Authorization: Bearer ${API_KEY}
      Content-Type: application/json
    queryParams:
      param1: value1
      param2: value2
    # disableSslVerification: false
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field**              |     **type**      | **required** | **description**                                                                                                                    |
|------------------------|:-----------------:|:------------:|------------------------------------------------------------------------------------------------------------------------------------|
| type                   |      string       |     true     | Must be "http".                                                                                                                    |
| baseUrl                |      string       |     true     | The base URL for the HTTP requests (e.g., `https://api.example.com`).                                                              |
| timeout                |      string       |    false     | The timeout for HTTP requests (e.g., "5s", "1m", refer to [ParseDuration][parse-duration-doc] for more examples). Defaults to 30s. |
| headers                | map[string]string |    false     | Default headers to include in the HTTP requests.                                                                                   |
| queryParams            | map[string]string |    false     | Default query parameters to include in the HTTP requests.                                                                          |
| disableSslVerification |       bool        |    false     | Disable SSL certificate verification. This should only be used for local development. Defaults to `false`.                         |

[parse-duration-doc]: https://pkg.go.dev/time#ParseDuration
