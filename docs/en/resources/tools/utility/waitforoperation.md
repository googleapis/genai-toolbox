---
title: "wait-for-operation"
type: docs
weight: 10
description: >
  Wait for a long-running operation to complete.
---

The `wait-for-operation` tool is a utility tool that waits for a long-running operation to complete. It does this by polling an operation status endpoint until the operation is finished using exponential backoff.

{{% notice info %}} 
This tool is intended for developer assistant workflows with human-in-the-loop and shouldn't be used for production agents.
{{% /notice %}}

## Example

```yaml
sources:
  my-http-source:
    kind: http
    baseUrl: https://api.example.com
tools:
  alloydb-operations-get:
    kind: wait-for-operation
    source: my-http-source
    method: GET
    path: /v1/projects/{{.projectId}}/locations/{{.locationId}}/operations/{{.operationId}}
    description: >
      Makes API call to check whether operation is done or not using exponential backoff. 
      if its still in create phase trigger it after 10 minutes timeout. 
      Print a message saying still not done.
    pathParams:
      - name: projectId
        type: string
        description: The dynamic path parameter
      - name: locationId
        type: string
        description: The dynamic path parameter
        default: us-central1
      - name: operationId
        type: string
        description: Operation status check for previous task
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                         |
| ----------- | :----------------------------------------: | :----------: | --------------------------------------------------------------------------------------- |
| kind        |                   string                   |     true     | Must be `wait-for-operation`.                                                           |
| source      |                   string                   |     true     | The name of the HTTP source to use for polling.                                         |
| method      |                   string                   |     true     | The HTTP method to use for the polling request (e.g., `GET`).                           |
| description |                   string                   |    false     | A description of the tool.                                                              |
| path        |                   string                   |     true     | The path to the operation status endpoint. You can use template parameters in the path. |
| pathParams  | [parameters](_index#specifying-parameters) |    false     | A list of parameters to substitute into the `path`.                                     |
