---
title: "dataplex-get-data-asset"
type: docs
weight: 2
description: >
  A "dataplex-get-data-asset" tool retrieve specific metadata regarding a Data Asset.
aliases:
  - /integrations/dataplex/tools/dataplex-get-data-asset/
---

## About

A `dataplex-get-data-asset` tool retrieves detailed metadata for a specific Data Asset in Knowledge Catalog (formerly known as Dataplex).

`dataplex-get-data-asset` requires the following parameter:

- `name` - The resource name of the Data Asset in the following form: `projects/{project}/locations/{location}/dataProducts/{dataProduct}/dataAssets/{dataAsset}`.

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
[dataplex-docs]: https://cloud.google.com/dataplex

## Example

```yaml
kind: tool
name: get_data_asset
type: dataplex-get-data-asset
source: my-dataplex-source
description: Use this tool to retrieve a Data Asset.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "dataplex-get-data-asset".                 |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
