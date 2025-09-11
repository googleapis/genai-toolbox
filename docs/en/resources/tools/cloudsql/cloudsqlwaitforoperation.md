---
title: "cloud-sql-wait-for-operation"
type: docs
weight: 10
description: >
  Wait for a long-running Cloud SQL operation to complete.
---

The `cloud-sql-wait-for-operation` tool is a utility tool that waits for a
long-running Cloud SQL operation to complete. It does this by polling the Cloud
SQL Admin API operation status endpoint until the operation is finished, using
exponential backoff.


## Example

```yaml
tools:
  cloudsql-operations-get:
    kind: cloud-sql-wait-for-operation
    source: my-cloud-sql-source
    description: "This will poll on operations API until the operation is done. For checking operation status we need projectId and operationId. Once instance is created give follow up steps on how to use the variables to bring data plane MCP server up in local and remote setup."
    delay: 1s
    maxDelay: 4m
    multiplier: 2
    maxRetries: 10
```

## Reference

| **field**   | **type** | **required** | **description**                                                                                                  |
| ----------- | :------: | :----------: | ---------------------------------------------------------------------------------------------------------------- |
| kind        |  string  |     true     | Must be "cloud-sql-wait-for-operation".                                                                          |
| source      |  string  |     true     | The name of a `cloud-sql-admin` source to use for authentication.                                                |
| description |  string  |    true      | A description of the tool.                                                                                       |
| delay       | duration |    false     | The initial delay between polling requests (e.g., `3s`). Defaults to 3 seconds.                                  |
| maxDelay    | duration |    false     | The maximum delay between polling requests (e.g., `4m`). Defaults to 4 minutes.                                  |
| multiplier  |  float   |    false     | The multiplier for the polling delay. The delay is multiplied by this value after each request. Defaults to 2.0. |
| maxRetries  |   int    |    false     | The maximum number of polling attempts before giving up. Defaults to 10.                                         |
