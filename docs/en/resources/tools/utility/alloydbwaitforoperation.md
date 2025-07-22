---
title: "alloydb-wait-for-operation"
type: docs
weight: 10
description: >
  Wait for a long-running AlloyDB operation to complete.
---

The `alloydb-wait-for-operation` tool is a utility tool that waits for a long-running AlloyDB operation to complete. It does this by polling the AlloyDB Admin API operation status endpoint until the operation is finished, using exponential backoff.

{{% notice info %}}
This tool is intended for developer assistant workflows with human-in-the-loop and shouldn't be used for production agents.
{{% /notice %}}

## Example

```yaml
  my-http-source:
    kind: http

tools:
  wait-for-alloydb-op:
    kind: alloydb-wait-for-operation
    source: my-google-source
    description: >
      Waits for an AlloyDB operation to complete.
```

## Reference

| **field**   | **type** | **required** | **description**                                                                                                                               |
| ----------- | :------: | :----------: | --------------------------------------------------------------------------------------------------------------------------------------------- |
| kind        |  string  |     true     | Must be `alloydb-wait-for-operation`.                                                                                                         |
| source      |  string  |     true     | The name of the HTTP source to use for polling. See the `http` source documentation for more details. The source must be of kind `http` and should be configured with Google authentication. |
| description |  string  |    false     | A description of the tool.                                                                                                                    |
| delay       | duration |    false     | The initial delay between polling requests (e.g., `3s`). Defaults to 3 seconds.                                                               |
| maxDelay    | duration |    false     | The maximum delay between polling requests (e.g., `4m`). Defaults to 4 minutes.                                                               |
| multiplier  |  float   |    false     | The multiplier for the polling delay. The delay is multiplied by this value after each request. Defaults to 2.0.                                |
| maxRetries  |   int    |    false     | The maximum number of polling attempts before giving up. Defaults to 10.                                                                      |

### Input Parameters

The tool takes the following parameters as input:

| **field**      | **type** | **required** | **description**                               |
| -------------- | :------: | :----------: | --------------------------------------------- |
| project        |  string  |     true     | The Google Cloud project ID.                  |
| location       |  string  |     true     | The location of the AlloyDB cluster.          |
| operation_id   |  string  |     true     | The ID of the operation to wait for.          |
