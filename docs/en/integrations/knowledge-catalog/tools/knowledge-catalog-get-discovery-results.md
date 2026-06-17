---
title: "dataplex-get-discovery-results"
type: docs
weight: 1
description: >
  Retrieves the final generated data discovery results (publishing metadata showing registered BigQuery tables, scanned file counts, and processed bytes) for a completed discovery scan.
aliases:
  - /integrations/dataplex/tools/dataplex-get-discovery-results/
---

## About

A `dataplex-get-discovery-results` tool retrieves the results of a completed Data Discovery scan.

WARNING: You must verify the execution run has succeeded (via `dataplex-get-run-status`) before calling this tool, otherwise the results will be empty.
CRITICAL: Access the results via the nested public fields `dataDiscoveryResult.bigqueryPublishing` and `dataDiscoveryResult.scanStatistics` inside the returned DataScan.


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

The `dataplex-get-discovery-results` tool accepts the following parameters:

| **field** | **type** | **required** | **description** |
| --------- | :------: | :----------: | --------------- |
| scanId | string | true | The unique ID of the Dataplex discovery scan (e.g. `nq-disc-12345`). |
| location | string | true | The Google Cloud region where the scan was created (e.g. `us-central1`). |

## Example

```yaml
kind: tool
name: get_discovery_results
type: dataplex-get-discovery-results
source: my-dataplex-source
description: Fetch results of a completed metadata discovery scan.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "dataplex-get-discovery-results".                   |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
