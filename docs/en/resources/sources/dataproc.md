---
title: "Dataproc Clusters"
type: docs
weight: 1
description: >
  Google Cloud Dataproc Clusters lets you provision and manage Apache Spark and Hadoop clusters.
---

## About

The [Dataproc
Clusters](https://cloud.google.com/dataproc/docs/concepts/overview) source
allows Toolbox to interact with Dataproc Clusters hosted on Google Cloud.

## Available Tools

- [`dataproc-get-cluster`](../tools/dataproc/dataproc-get-cluster.md)
  Get a specific Dataproc cluster.
- [`dataproc-list-clusters`](../tools/dataproc/dataproc-list-clusters.md)
  List and filter Dataproc clusters.
- [`dataproc-get-job`](../tools/dataproc/dataproc-get-job.md)
  Get a specific Dataproc job.
- [`dataproc-list-jobs`](../tools/dataproc/dataproc-list-jobs.md)
  List and filter Dataproc jobs.

## Requirements

### IAM Permissions

Dataproc uses [Identity and Access Management
(IAM)](https://cloud.google.com/dataproc/docs/concepts/iam/iam) to control user
and group access to Dataproc resources.

Toolbox will use your [Application Default Credentials
(ADC)](https://cloud.google.com/docs/authentication#adc) to authorize and
authenticate when interacting with Dataproc. When using this method, you need to
ensure the IAM identity associated with your ADC has the correct
[permissions](https://cloud.google.com/dataproc/docs/concepts/iam/iam)
for the actions you intend to perform. Common roles include
`roles/dataproc.editor` or `roles/dataproc.viewer`. Follow this
[guide](https://cloud.google.com/docs/authentication/provide-credentials-adc) to
set up your ADC.

## Example

```yaml
kind: sources
name: my-dataproc-source
type: dataproc
project: my-project
region: us-central1
```

## Reference

| **field** | **type** | **required** | **description**                                    |
| --------- | :------: | :----------: | -------------------------------------------------- |
| type      |  string  |     true     | Must be "dataproc".                                |
| project   |  string  |     true     | ID of the GCP project with Dataproc resources.     |
| region    |  string  |     true     | Region containing Dataproc resources.            |
