---
title: "alloydb-list-instances"
type: docs
weight: 1
description: >
  The "alloydb-list-instances" tool lists the AlloyDB instances for a given project, cluster and location.
aliases:
- /resources/tools/alloydb-list-instances
---

## About

The `alloydb-list-instances` tool retrieves AlloyDB instance information for all or specified clusters and locations in a given project.

`alloydb-list-instances` tool lists the detailed information of AlloyDB instances (instance name, type, IP address, state, configuration, etc) for a given project, cluster and location. The tool takes the following input parameters:
	
| Parameter  | Type   | Description                                                                              | Required |
| :--------- | :----- | :--------------------------------------------------------------------------------------- | :------- |
| `projectId`  | string | The GCP project ID to list instances for.                                                 | Yes      |
| `clusterId` | string | The ID of the cluster to list instances from. Use '-' to get results for all clusters. Default: `-`.| No       |
| `locationId` | string | The location of the cluster (e.g., 'us-central1'). Use '-' to get results for all locations. Default: `-`.| No       |
> **Note**
> This tool does not have a `source` and authenticates using the environment's
[Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials).

## Example

```yaml
tools:
  alloydb_list_instances:
    kind: alloydb-list-instances
    description: Use this tool to list all AlloyDB instances for a given project, cluster and location.
```
## Reference
| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be alloydb-list-instances.                                                                  |                                               |
| description |                   string                   |     true     | Description of the tool that is passed to the agent.                                             |