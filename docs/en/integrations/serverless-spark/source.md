---
title: "Serverless for Apache Spark Source"
linkTitle: "Source"
type: docs
weight: 1
description: >
  Google Cloud Serverless for Apache Spark lets you run Spark workloads without requiring you to provision and manage your own Spark cluster.
no_list: true
---

## About

The [Serverless for Apache
Spark](https://cloud.google.com/dataproc-serverless/docs/overview) source allows
Toolbox to interact with Spark batches hosted on Google Cloud Serverless for
Apache Spark.



## Available Tools

{{< list-tools >}}

## Requirements

### IAM Permissions

Serverless for Apache Spark uses [Identity and Access Management
(IAM)](https://cloud.google.com/bigquery/docs/access-control) to control user
and group access to serverless Spark resources like batches and sessions.

Toolbox will use your [Application Default Credentials
(ADC)](https://cloud.google.com/docs/authentication#adc) to authorize and
authenticate when interacting with Google Cloud Serverless for Apache Spark.
When using this method, you need to ensure the IAM identity associated with your
ADC has the correct
[permissions](https://cloud.google.com/dataproc-serverless/docs/concepts/iam)
for the actions you intend to perform. Common roles include
`roles/dataproc.serverlessEditor` (which includes permissions to run batches) or
`roles/dataproc.serverlessViewer`. Follow this
[guide](https://cloud.google.com/docs/authentication/provide-credentials-adc) to
set up your ADC.

## Example

```yaml
kind: source
name: my-serverless-spark-source
type: serverless-spark
project: my-project-id
location: us-central1
```

## Reference

| **field** | **type** | **required** | **description**                                                   |
| --------- | :------: | :----------: | ----------------------------------------------------------------- |
| type      |  string  |     true     | Must be "serverless-spark".                                       |
| project   |  string  |     true     | ID of the GCP project with Serverless for Apache Spark resources. |
| location  |  string  |     true     | Location containing Serverless for Apache Spark resources.        |
