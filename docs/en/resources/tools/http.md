---
title: "http"
type: docs
weight: 1
description: > 
  A "http" tool sends out an HTTP request to an HTTP endpoint.
---


## About

The `http` tool allows you to make HTTP requests to APIs to retrieve data.
An HTTP request is the method by which a client communicates with a server to retrieve or manipulate resources.
Toolbox allows you to configure the request URL, method, headers, query parameters, and the request body for an HTTP Tool.

### URL

An HTTP request URL identifies the targetthe client wants to access.
Toolbox composes the request URL from the HTTP Source's `baseUrl` and the HTTP Tool's `path`.
For example, the following config allows you to reach different paths of the same server using multiple Tools:

```yaml
sources:
    my-http-source:
        kind: http
        BaseUrl: https://api.example.com

tools:
    my-post-tool:
        kind: http
        source: my-http-source
        method: POST
        path: /update
        description: Tool to update information to the example API

    my-get-tool:
        kind: http
        source: my-http-source
        method: GET
        path: /search
        description: Tool to search information from the example API

```

### Headers

An HTTP request header is a key-value pair sent by a client to a server, providing additional information about the request, such as the client's preferences, the request body content type, and other metadata.
Headers specified by the HTTP Tool are combined with its HTTP Source headers for the resulting HTTP request, and overrides the Source headers in case of conflict.
The HTTP Tool allows you to specify headers in two different way:

- Static headers can be specified using the `headers` field, and will be the same for every invocation:

```yaml
my-http-tool:
    kind: http
    source: my-http-source
    method: GET
    path: /search
    description: Tool to search data from API
    headers:
      Authorization: API_KEY
      Content-Type: application/json
```

- Dynamic headers can be specified as parameters in the `headerParams` field. The `name` of the `headerParams` will be used as the header key, and the value is determined by the LLM input upon Tool invocation:

```yaml
my-http-tool:
    kind: http
    source: my-http-source
    method: GET
    path: /search
    description: some description
    headerParams:
      - name: Content-Type # Example LLM input: "application/json"
        description: request content type
        type: string
```

### Query parameters

- Static request query parameters should be specified in the `path` as part of the URL itself:

```yaml
my-http-tool:
    kind: http
    source: my-http-source
    method: GET
    path: /search?language=en&id=1
    description: Tool to search for item with ID 1 in English
```

- Dynamic request query parameters should be specified as parameters in the `headerParams` section:

```yaml
my-http-tool:
    kind: http
    source: my-http-source
    method: GET
    path: /search
    description: Tool to search for item with ID
    queryParams:
      - name: id
        description: iteam ID
        type: integer
```

### Request body

The request body payload is a string that supports parameter replacement with the `$` plus vriable name as the placeholders.
For example, `$id` will be replaced by the value of the parameter with the name `id` and `$age` by the value of the parameter with the name `age`. The parameter values will be populated into the request body payload upon Tool invocation.
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
      "age": {{.age}}
      "city": {{.city}}
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
      Content-Type: application/json
    headerParams:
      - name: Language
        description: language string
        type: string
```

## Reference

| **field**    |                  **type**                  | **required** | **description**                                                                                                                                                                                                                             |
|--------------|:------------------------------------------:|:------------:|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| kind         |                   string                   |     true     | Must be "http".                                                                                                                                                                                                                             |
| source       |                   string                   |     true     | Name of the source the HTTP request should be sent to.                                                                                                                                                                                      |
| description  |                   string                   |     true     | Description of the tool that is passed to the LLM.                                                                                                                                                                                          |
| path         |                   string                   |     true     | The path of the HTTP request.                                                                                                                                                                                                               |
| method       |                   string                   |     true     | The HTTP method to use (e.g., GET, POST, PUT, DELETE).                                                                                                                                                                                      |
| headers      |             map[string]string              |    false     | A map of headers to include in the HTTP request (overrides source headers).                                                                                                                                                                 |
| requestBody  |                   string                   |    false     | The request body payload. Use [go template](https://pkg.go.dev/text/template) with the parameter name as the placeholder (e.g., `{{.id}}` will be replaced with the value of the parameter that has name `id` in the `bodyParams` section). |
| queryParams  | [parameters](_index#specifying-parameters) |    false     | List of [parameters](_index#specifying-parameters) that will be inserted into the query string.                                                                                                                                             |
| bodyParams   | [parameters](_index#specifying-parameters) |    false     | List of [parameters](_index#specifying-parameters) that will be inserted into the request body payload.                                                                                                                                     |
| headerParams | [parameters](_index#specifying-parameters) |    false     | List of [parameters](_index#specifying-parameters) that will be inserted as the request headers.                                                                                                                                            |
