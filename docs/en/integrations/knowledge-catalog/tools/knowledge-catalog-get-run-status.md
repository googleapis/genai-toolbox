---
title: "dataplex-get-run-status"
type: docs
weight: 1
description: >
  Retrieves the execution status of the background job run (DataScanJob) for a specified Dataplex scan.
aliases:
  - /integrations/dataplex/tools/dataplex-get-run-status/
---

## About

A `dataplex-get-run-status` tool retrieves the execution status of the latest background job run for a specified scan.

Use this tool to poll the progress of the insights, profiling, discovery, or quality scan execution. Wait until the returned `state` is `SUCCEEDED` before fetching results. Typical execution takes 2-5 minutes. If the state is `FAILED`, check the error details.


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

The `dataplex-get-run-status` tool accepts the following parameters:

| **field** | **type** | **required** | **description** |
| --------- | :------: | :----------: | --------------- |
| scanId | string | true | The unique ID of the Dataplex scan template (e.g. `nq-prof-12345`). |
| location | string | true | The Google Cloud region where the scan was created (e.g. `us-central1`). |
| jobId | string | false | Optional. A specific job run ID. If omitted, returns status for the latest job run. |

## Example

```yaml
kind: tool
name: get_run_status
type: dataplex-get-run-status
source: my-dataplex-source
description: Monitor the background execution run of a Dataplex scan.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "dataplex-get-run-status".                   |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
