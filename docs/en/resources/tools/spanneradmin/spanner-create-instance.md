---
title: spanner-create-instance
type: docs
weight: 2
description: "Create a Cloud Spanner instance."
---

The `spanner-create-instance` tool creates a new Cloud Spanner instance in a
specified Google Cloud project.

{{< notice info >}}
This tool uses the `spanner-admin` source.
{{< /notice >}}

## Configuration

Here is an example of how to configure the `spanner-create-instance` tool in
your `tools.yaml` file:

```yaml
sources:
  my-spanner-admin-source:
    kind: spanner-admin

tools:
  create_my_spanner_instance:
    kind: spanner-create-instance
    source: my-spanner-admin-source
    description: "Creates a Spanner instance."
```

## Parameters

The `spanner-create-instance` tool has the following parameters:

| **field**       | **type** | **required** | **description**                                                                      |
| --------------- | :------: | :----------: | ------------------------------------------------------------------------------------ |
| project         |  string  |     true     | The Google Cloud project ID.                                                         |
| instanceId      |  string  |     true     | The ID of the instance to create.                                                    |
| displayName     |  string  |     true     | The display name of the instance.                                                    |
| config          |  string  |     true     | The instance configuration (e.g., `regional-us-central1`).                           |
| nodeCount       | integer  |     true     | The number of nodes. Mutually exclusive with `processingUnits` (one must be 0).      |
| processingUnits | integer  |     true     | The number of processing units. Mutually exclusive with `nodeCount` (one must be 0). |
| edition         |  string  |    false     | The edition of the instance (`STANDARD`, `ENTERPRISE`, `ENTERPRISE_PLUS`).           |

## Reference

| **field**   | **type** | **required** | **description**                                              |
| ----------- | :------: | :----------: | ------------------------------------------------------------ |
| kind        |  string  |     true     | Must be `spanner-create-instance`.                           |
| source      |  string  |     true     | The name of the `spanner-admin` source to use for this tool. |
| description |  string  |    false     | A description of the tool that is passed to the agent.       |
