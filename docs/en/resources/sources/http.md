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

## Example

```yaml
sources:
  my-http-source:
    kind: http
    baseUrl: https://api.example.com/data
    timeout: 10s # default to 30s
    headers:
      Authorization: Bearer YOUR_API_TOKEN
      Content-Type: application/json
    queryParams:
      param1: value1
      param2: value2
```

## Reference

| **field**   |     **type**      | **required** | **description**                                                                                                                   |
|-------------|:-----------------:|:------------:|-----------------------------------------------------------------------------------------------------------------------------------|
| kind        |      string       |     true     | Must be "http".                                                                                                                   |
| baseUrl     |      string       |     true     | The base URL for the HTTP requests (e.g., `https://api.example.com`).                                                             |
| timeout     |      string       |    false     | The timeout for HTTP requests (e.g., "5s", "1m", refer to [ParseDuration][parse-duration-doc] for more examples). Defaults to 30s. |
| headers     | map[string]string |    false     | Default headers to include in the HTTP requests.                                                                                  |
| queryParams | map[string]string |    false     | Default query parameters to include in the HTTP requests.                                                                         |

[parse-duration-doc]: https://pkg.go.dev/time#ParseDuration
