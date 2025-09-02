---
title: "cloudsql-wait-for-operation"
type: docs
weight: 10
description: >
  Wait for a long-running Cloud SQL operation to complete.
---

The `cloudsql-wait-for-operation` tool is a utility tool that waits for a
long-running Cloud SQL operation to complete. It does this by polling the Cloud
SQL Admin API operation status endpoint until the operation is finished, using
exponential backoff.

{{< notice info >}}
This tool is intended for developer assistant workflows with human-in-the-loop
and shouldn't be used for production agents.
{{< /notice >}}

{{< notice info >}}
This tool does not have a `source` and authenticates using the environment's
[Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials).
{{< /notice >}}

## Example

```yaml
tools:
  cloudsql-operations-get:
    kind: cloudsql-wait-for-operation
    description: "This will poll on operations API until the operation is done. For checking operation status we need projectId and operationId. Once instance is created give follow up steps on how to use the variables to bring data plane MCP server up in local and remote setup."
    delay: 1s
    maxDelay: 4m
    multiplier: 2
    maxRetries: 10
```

## Reference

| **field**   | **type** | **required** | **description**                                                                                                  |
| ----------- | :------: | :----------: | ---------------------------------------------------------------------------------------------------------------- |
| kind        |  string  |     true     | Must be "cloudsql-wait-for-operation".                                                                           |
| description |  string  |    true      | A description of the tool.                                                                                       |
| delay       | duration |    false     | The initial delay between polling requests (e.g., `3s`). Defaults to 3 seconds.                                  |
| maxDelay    | duration |    false     | The maximum delay between polling requests (e.g., `4m`). Defaults to 4 minutes.                                  |
| multiplier  |  float   |    false     | The multiplier for the polling delay. The delay is multiplied by this value after each request. Defaults to 2.0. |
| maxRetries  |   int    |    false     | The maximum number of polling attempts before giving up. Defaults to 10.                                         |
