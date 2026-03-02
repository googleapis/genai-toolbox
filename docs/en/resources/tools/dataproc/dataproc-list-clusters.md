---
title: "dataproc-list-clusters"
type: docs
weight: 1
description: >
  A "dataproc-list-clusters" tool returns a list of Dataproc clusters from the source.
aliases:
  - /resources/tools/dataproc-list-clusters
---

## About

A `dataproc-list-clusters` tool returns a list of Dataproc clusters from a
Google Cloud Dataproc source. It's compatible with the
following sources:

- [dataproc](../../sources/dataproc.md)

`dataproc-list-clusters` accepts the following parameters:

- **`filter`** (optional): A filter expression to limit the clusters returned.
  Filters are case sensitive and may contain multiple clauses combined with
  logical operators (AND only). Supported fields are `status.state`, `clusterName`,
  and `labels`. For example: `status.state = ACTIVE AND clusterName = mycluster`.
  Supported `status.state` values are: `ACTIVE`, `INACTIVE`, `CREATING`, `RUNNING`,
  `ERROR`, `DELETING`, `UPDATING`, `STOPPING`, `STOPPED`.
- **`pageSize`** (optional): The maximum number of clusters to return in a single
  page.
- **`pageToken`** (optional): A page token, received from a previous call, to
  retrieve the next page of results. Defaults to `20`.

The tool gets the `project` and `region` from the source configuration.

## Example

```yaml
kind: tools
name: list_clusters
type: dataproc-list-clusters
source: my-dataproc-source
description: Use this tool to list and filter Dataproc clusters.
```

## Response Format

```json
{
  "clusters": [
    {
      "name": "projects/my-project/regions/us-central1/clusters/cluster-1",
      "uuid": "a1b2c3d4-e5f6-7890-1234-567890abcdef",
      "state": "RUNNING",
      "createTime": "2023-10-27T10:00:00Z",
      "consoleUrl": "https://console.cloud.google.com/dataproc/clusters/cluster-1/monitoring?region=us-central1&project=my-project",
      "logsUrl": "https://console.cloud.google.com/logs/viewer?advancedFilter=resource.type%3D%22cloud_dataproc_cluster%22%0Aresource.labels.project_id%3D%22my-project%22%0Aresource.labels.region%3D%22us-central1%22%0Aresource.labels.cluster_name%3D%22cluster-1%22%0Aresource.labels.cluster_uuid%3D%22a1b2c3d4-e5f6-7890-1234-567890abcdef%22&project=my-project&resource=cloud_dataproc_cluster%2Fcluster_name%2Fcluster-1"
    },
    {
      "name": "projects/my-project/regions/us-central1/clusters/cluster-2",
      "uuid": "b2c3d4e5-f6a7-8901-2345-678901bcdefa",
      "state": "ERROR",
      "createTime": "2023-10-27T11:30:00Z",
      "consoleUrl": "https://console.cloud.google.com/dataproc/clusters/cluster-2/monitoring?region=us-central1&project=my-project",
      "logsUrl": "https://console.cloud.google.com/logs/viewer?advancedFilter=resource.type%3D%22cloud_dataproc_cluster%22%0Aresource.labels.project_id%3D%22my-project%22%0Aresource.labels.region%3D%22us-central1%22%0Aresource.labels.cluster_name%3D%22cluster-2%22%0Aresource.labels.cluster_uuid%3D%22b2c3d4e5-f6a7-8901-2345-678901bcdefa%22&project=my-project&resource=cloud_dataproc_cluster%2Fcluster_name%2Fcluster-2"
    }
  ],
  "nextPageToken": "abcd1234"
}
```

## Reference

| **field**    | **type** | **required** | **description**                                    |
| ------------ | :------: | :----------: | -------------------------------------------------- |
| type         |  string  |     true     | Must be "dataproc-list-clusters".                  |
| source       |  string  |     true     | Name of the source the tool should use.            |
| description  |  string  |     true     | Description of the tool that is passed to the LLM. |
| authRequired | string[] |    false     | List of auth services required to invoke this tool |
