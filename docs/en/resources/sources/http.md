---
title: "HTTP Source"
linkTitle: "HTTP"
type: docs
weight: 1
description: >
  The HTTP Source enables the Gen AI Toolbox to retrieve data from HTTP
  endpoints.
---

## About

The HTTP Source allows the Gen AI Toolbox to retrieve data from arbitrary HTTP
endpoints. This enables Generative AI applications to access data from web APIs
and other HTTP-accessible resources.

## Example

```yaml
sources:
  my-http-source:
    kind: "http"
    baseUrl: "https://api.example.com/data"
    timeout: "10s"
    headers:
      Authorization: "Bearer YOUR_API_TOKEN"
      Content-Type: "application/json"
    queryParams:
      param1: value1
      param2: value2
```

## Reference

| **field** | **type** | **required** | **description**                                                                           |
|-----------|:--------:|:------------:|-------------------------------------------------------------------------------------------|
| kind | string | true | Must be "http". |
| name | string | true | A unique name for this HTTP source. |
| baseUrl | string | true | The base URL for the HTTP requests (e.g., "<https://api.example.com>"). |
| timeout | string | true | The timeout for HTTP requests (e.g., "5s", "1m", refer to this [doc](https://pkg.go.dev/time#ParseDuration) for more info)  . |
| headers | map[string]string | false | Default headers to include in the HTTP requests.  |
