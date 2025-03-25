---
title: "http-json"
type: docs
weight: 1
description: > 
  A "http-json" tool sends out an HTTP request with a JSON request body to an HTTP endpoint.
---


## About

The `http-json` tool allows you to make HTTP requests with JSON content type to APIs and retrieve data.
Both static and dynamic parameters are supported as part of the request.

- Static parameters stay the same for every Tool invocation.
- Dynamic parameters are populated upon Tool invocation by the LLM inputs.

### Request headers

- Static request headers should be specified as maps in the `headers` field.
- Dynamic request headers should be specified as parameters in the `headerParams` section.

### Request query parameters

- Static request query parameters should be specified in the `path` as part of the URL itself (e.g., "/endpoint1?language=en").
- Dynamic request query parameters should be specified as parameters in the `headerParams` section.

### Request body

The request body payload is a string that supports parameter replacement with the `$` plus vriable name as the placeholders.
For example, `$id` will be replaced by the value of the parameter with the name `id` and `$age` by the value of the parameter with the name `age`. The parameter values will be JSON encoded and then populated into the request body payload.
Specify replacement parameters in the `bodyParams` section.

## Example

```yaml
my-http-tool:
    kind: http
    source: my-http-source
    method: GET
    path: /search
    description: some description
    authRequired:
      - my-google-auth-service
      - other-auth-service
    queryParams:
      - name: country
        description: some description
        type: string
    requestBody: |
      {
      "age": $age
      "city": $city
      "food": $food
      }
    bodyParams:
      - name: age
        description: age number
        type: integer
      - name: city
        description: city string
        type: string
    headers:
      Authorization: API_KEY
    headerParams:
      - name: Language
        description: language string
        type: string
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "http-json".                                               |
| source      |                   string                   |     true     | Name of the source the HTTP request should be sent to.                                 |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                      |
| path        |                   string                   |     true     | The path of the HTTP request.      |
| method      |                   string                   |     true     | The HTTP method to use (e.g., GET, POST, PUT, DELETE).|
| headers     |              map[string]string             |    false     | A map of headers to include in the HTTP request (overrides source headers).            |
| requestBody |                   string                   |    false     | The request body payload. Use `$` with the parameter name as the placeholder (e.g., `$id` will be replaced with the value of the parameter that has name `id` in the `bodyParams` section). Values will be JSON encoded before the replacement, therefore no need to wrap the string values with quotes.|
| queryParams | [parameters](_index#specifying-parameters) |    false     | List of [parameters](_index#specifying-parameters) that will be inserted into the query string (overrides source `queryParams` in case of conflict).|
| bodyParams  | [parameters](_index#specifying-parameters) |    false     | List of [parameters](_index#specifying-parameters) that will be inserted into the request body payload.   |
| headerParams| [parameters](_index#specifying-parameters) |    false     | List of [parameters](_index#specifying-parameters) that will be inserted as the request headers (overrides source and tool `headers` in case of conflict).|
