---
title: "dataproc-list-jobs"
type: docs
weight: 1
description: >
  A "dataproc-list-jobs" tool returns a list of Dataproc jobs from the source.
aliases:
  - /resources/tools/dataproc-list-jobs
---

## About

A `dataproc-list-jobs` tool returns a list of Dataproc jobs from a
Google Cloud Dataproc source. It's compatible with the
following sources:

- [dataproc](../../sources/dataproc.md)

`dataproc-list-jobs` accepts the following parameters:

- **`filter`** (optional): A filter expression to limit the jobs returned.
  Filters are case sensitive and may contain multiple clauses combined with
  logical operators (AND only). Supported fields are `status.state` and `labels`.
  For example: `status.state = RUNNING AND labels.env = production`.
  Supported `status.state` values are: `PENDING`, `RUNNING`, `CANCEL_PENDING`,
  `JOB_STATE_CANCELLED`, `DONE`, `ERROR`, `ATTEMPT_FAILURE`.
- **`jobStateMatcher`** (optional): Specifies if the job state matcher should match
  ALL jobs, only ACTIVE jobs, or only NON_ACTIVE jobs. Defaults to ALL.
  Supported values: `ALL`, `ACTIVE`, `NON_ACTIVE`.
- **`pageSize`** (optional): The maximum number of jobs to return in a single
  page.
- **`pageToken`** (optional): A page token, received from a previous call, to
  retrieve the next page of results.

The tool gets the `project` and `region` from the source configuration.

## Example

```yaml
kind: tools
name: list_jobs
type: dataproc-list-jobs
source: my-dataproc-source
description: Use this tool to list and filter Dataproc jobs.
```

## Response Format

```json
{
  "jobs": [
    {
      "id": "job-1",
      "status": "DONE",
      "subStatus": "HOURS",
      "startTime": "2023-10-27T10:00:00Z",
      "endTime": "2023-10-27T10:05:00Z",
      "clusterName": "cluster-1",
      "consoleUrl": "https://console.cloud.google.com/dataproc/jobs/job-1?region=us-central1&project=my-project",
      "logsUrl": "https://console.cloud.google.com/logs/viewer?advancedFilter=resource.type%3D%22cloud_dataproc_cluster%22%0Aresource.labels.project_id%3D%22my-project%22%0Aresource.labels.region%3D%22us-central1%22%0Aresource.labels.cluster_name%3D%22cluster-1%22%0Alabels.job_id%3D%22job-1%22&project=my-project&resource=cloud_dataproc_cluster%2Fcluster_name%2Fcluster-1"
    },
    {
      "id": "job-2",
      "status": "RUNNING",
      "startTime": "2023-10-27T10:10:00Z",
      "clusterName": "cluster-1",
      "consoleUrl": "https://console.cloud.google.com/dataproc/jobs/job-2?region=us-central1&project=my-project",
      "logsUrl": "https://console.cloud.google.com/logs/viewer?advancedFilter=resource.type%3D%22cloud_dataproc_cluster%22%0Aresource.labels.project_id%3D%22my-project%22%0Aresource.labels.region%3D%22us-central1%22%0Aresource.labels.cluster_name%3D%22cluster-1%22%0Alabels.job_id%3D%22job-2%22&project=my-project&resource=cloud_dataproc_cluster%2Fcluster_name%2Fcluster-1"
    }
  ],
  "nextPageToken": "abcd1234"
}
```

## Reference

| **field**    | **type** | **required** | **description**                                    |
| ------------ | :------: | :----------: | -------------------------------------------------- |
| type         |  string  |     true     | Must be "dataproc-list-jobs".                      |
| source       |  string  |     true     | Name of the source the tool should use.            |
| description  |  string  |     true     | Description of the tool that is passed to the LLM. |
| authRequired | string[] |    false     | List of auth services required to invoke this tool |
