---
title: "dataproc-get-cluster"
type: docs
weight: 1
description: >
  A "dataproc-get-cluster" tool retrieves a specific Dataproc cluster from the source.
aliases:
  - /resources/tools/dataproc-get-cluster
---

## About

A `dataproc-get-cluster` tool retrieves a specific Dataproc cluster from a
Google Cloud Dataproc source. It's compatible with the
following sources:

- [dataproc](../../sources/dataproc.md)

`dataproc-get-cluster` accepts the following parameters:

- **`clusterName`** The short name of the cluster to retrieve. e.g. for
  `projects/my-project/regions/us-central1/clusters/my-cluster`, pass
  `my-cluster`.

The tool gets the `project` and `region` from the source configuration.

## Example

```yaml
kind: tools
name: get_cluster
type: dataproc-get-cluster
source: my-dataproc-source
description: Use this tool to get details of a Dataproc cluster.
```

## Response Format

```json
{
  "cluster": {
    "name": "projects/my-project/regions/us-central1/clusters/my-cluster",
    "uuid": "a1b2c3d4-e5f6-7890-1234-567890abcdef",
    "state": "RUNNING",
    "createTime": "2023-10-27T10:00:00Z",
  },
  "consoleUrl": "https://console.cloud.google.com/dataproc/clusters/my-cluster/monitoring?region=us-central1&project=my-project",
  "logsUrl": "https://console.cloud.google.com/logs/viewer?advancedFilter=resource.type%3D%22cloud_dataproc_cluster%22%0Aresource.labels.project_id%3D%22my-project%22%0Aresource.labels.region%3D%22us-central1%22%0Aresource.labels.cluster_name%3D%22my-cluster%22%0Aresource.labels.cluster_uuid%3D%22a1b2c3d4-e5f6-7890-1234-567890abcdef%22&project=my-project&resource=cloud_dataproc_cluster%2Fcluster_name%2Fmy-cluster"
}
```

## Reference

| **field**    | **type** | **required** | **description**                                    |
| ------------ | :------: | :----------: | -------------------------------------------------- |
| type         |  string  |     true     | Must be "dataproc-get-cluster".                    |
| source       |  string  |     true     | Name of the source the tool should use.            |
| description  |  string  |     true     | Description of the tool that is passed to the LLM. |
| authRequired | string[] |    false     | List of auth services required to invoke this tool |
