---
title: alloydb-wait-for-operation
type: docs
weight: 10
description: "Wait for a long-running AlloyDB operation to complete.\n"
---

The `alloydb-wait-for-operation` tool is a utility tool that waits for a
long-running AlloyDB operation to complete. It does this by polling the AlloyDB
Admin API operation status endpoint until the operation is finished, using
exponential backoff. It is compatible with
[alloydb-admin](../../sources/alloydb-admin.md) source.

| Parameter   | Type   | Description                                          | Required |
| :---------- | :----- | :--------------------------------------------------- | :------- |
| `project`   | string | The GCP project ID.                                  | Yes      |
| `location`  | string | The location of the operation (e.g., 'us-central1'). | Yes      |
| `operation` | string | The ID of the operation to wait for.                 | Yes      |

{{< notice info >}}
This tool is intended for developer assistant workflows with human-in-the-loop
and shouldn't be used for production agents.
{{< /notice >}}

## Example

```yaml
tools:
  wait_for_operation:
    kind: alloydb-wait-for-operation
    source: my-alloydb-admin-source
    description: "This will poll on operations API until the operation is done. For checking operation status we need projectId, locationID and operationId. Once instance is created give follow up steps on how to use the variables to bring data plane MCP server up in local and remote setup."
    delay: 1s
    maxDelay: 4m
    multiplier: 2
    maxRetries: 10
```

## Reference

| **field**   | **type** | **required** | **description**                                                                                                  |
| ----------- | :------: | :----------: | ---------------------------------------------------------------------------------------------------------------- |
| kind        |  string  |     true     | Must be "alloydb-wait-for-operation".                                                                            |
| source      |  string  |     true     | The name of a `alloydb-admin` source to use for authentication.                                                  |
| description |  string  |     false    | A description of the tool.                                                                                       |
| delay       | duration |     false    | The initial delay between polling requests (e.g., `3s`). Defaults to 3 seconds.                                  |
| maxDelay    | duration |     false    | The maximum delay between polling requests (e.g., `4m`). Defaults to 4 minutes.                                  |
| multiplier  |   float  |     false    | The multiplier for the polling delay. The delay is multiplied by this value after each request. Defaults to 2.0. |
| maxRetries  |    int   |     false    | The maximum number of polling attempts before giving up. Defaults to 10.                                         |
