---
title: "dataproc-get-job"
type: docs
weight: 1
description: >
  A "dataproc-get-job" tool retrieves a specific Dataproc job from the source.
aliases:
  - /resources/tools/dataproc-get-job
---

## About

A `dataproc-get-job` tool retrieves a specific Dataproc job from a
Google Cloud Dataproc source. It's compatible with the
following sources:

- [dataproc](../../sources/dataproc.md)

`dataproc-get-job` accepts the following parameters:

- **`jobId`** The job ID, e.g. for
  `projects/my-project/regions/us-central1/jobs/my-job`, pass
  `my-job`.

The tool gets the `project` and `region` from the source configuration.

## Example

```yaml
kind: tools
name: get_job
type: dataproc-get-job
source: my-dataproc-source
description: Use this tool to get details of a Dataproc job.
```

## Response Format

```json
{
  "job": {
    "reference": {
      "projectId": "my-project",
      "jobId": "my-job"
    },
    "placement": {
      "clusterName": "my-cluster",
      "clusterUuid": "a1b2c3d4-e5f6-7890-1234-567890abcdef"
    },
    ...
  },
  "consoleUrl": "https://console.cloud.google.com/dataproc/jobs/my-job?region=us-central1&project=my-project",
  "logsUrl": "https://console.cloud.google.com/logs/viewer?advancedFilter=resource.type%3D%22cloud_dataproc_cluster%22%0Aresource.labels.project_id%3D%22my-project%22%0Aresource.labels.region%3D%22us-central1%22%0Aresource.labels.cluster_name%3D%22my-cluster%22%0Alabels.job_id%3D%22my-job%22&project=my-project&resource=cloud_dataproc_cluster%2Fcluster_name%2Fmy-cluster"
}
```

## Reference

| **field**    | **type** | **required** | **description**                                    |
| ------------ | :------: | :----------: | -------------------------------------------------- |
| type         |  string  |     true     | Must be "dataproc-get-job".                        |
| source       |  string  |     true     | Name of the source the tool should use.            |
| description  |  string  |     true     | Description of the tool that is passed to the LLM. |
| authRequired | string[] |    false     | List of auth services required to invoke this tool |
