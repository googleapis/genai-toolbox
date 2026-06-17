---
title: "dataplex-discover-metadata"
type: docs
weight: 1
description: >
  Creates a new Dataplex Data Discovery scan template for a specified Cloud Storage bucket and triggers the initial asynchronous execution run to crawl files, infer schemas, and register tables in BigQuery.
aliases:
  - /integrations/dataplex/tools/dataplex-discover-metadata/
---

## About

A `dataplex-discover-metadata` tool triggers a new Data Discovery scan to automatically crawl GCS directories, infer schemas/partitions, and publish them as BigQuery tables.

Since scan template creation is asynchronous, this tool returns an LRO name. You must poll `dataplex-get-operation` with this ID until it is done, extract the `scanId`, and poll `dataplex-get-run-status` with the `scanId` until the job is `SUCCEEDED` before calling `dataplex-get-discovery-results` to fetch results.


## Compatible Sources

{{< compatible-sources >}}

## Requirements

### IAM Permissions

Knowledge Catalog uses [Identity and Access Management (IAM)][iam-overview] to control
user and group access to Knowledge Catalog resources. Toolbox will use your
[Application Default Credentials (ADC)][adc] to authorize and authenticate when
interacting with [Knowledge Catalog][dataplex-docs].

In addition to [setting the ADC for your server][set-adc], you need to ensure
the IAM identity has been given the correct IAM permissions for the tasks you
intend to perform. See [Knowledge Catalog IAM permissions][iam-permissions]
and [Knowledge Catalog IAM roles][iam-roles] for more information on
applying IAM permissions and roles to an identity.

[iam-overview]: https://cloud.google.com/dataplex/docs/iam-and-access-control
[adc]: https://cloud.google.com/docs/authentication#adc
[set-adc]: https://cloud.google.com/docs/authentication/provide-credentials-adc
[iam-permissions]: https://cloud.google.com/dataplex/docs/iam-permissions
[iam-roles]: https://cloud.google.com/dataplex/docs/iam-roles

## Parameters

The `dataplex-discover-metadata` tool accepts the following parameters:

| **field** | **type** | **required** | **description** |
| --------- | :------: | :----------: | --------------- |
| resourcePath | string | true | The resource path of the target Cloud Storage bucket (format: `//storage.googleapis.com/{bucket_name}`). |
| location | string | true | The Google Cloud region where the scan should be executed (e.g. `us-central1`). |

## Example

```yaml
kind: tool
name: discover_metadata
type: dataplex-discover-metadata
source: my-dataplex-source
description: Trigger a new metadata discovery scan.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "dataplex-discover-metadata".                   |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
