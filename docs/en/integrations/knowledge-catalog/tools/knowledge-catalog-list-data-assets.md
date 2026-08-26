---
title: "dataplex-list-data-assets"
type: docs
weight: 1
description: >
  A "dataplex-list-data-assets" tool allows to list Data Assets under a Data Product.
---

## About

A `dataplex-list-data-assets` tool retrieves a list of Data Assets associated with a specific Data Product in Knowledge Catalog (formerly known as Dataplex).

View the [Data Products guide][guide] for more information.

[guide]: https://docs.cloud.google.com/dataplex/docs/data-products-overview

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

## Parameters

The `dataplex-list-data-assets` tool has the following parameters:

| **field**     | **type** | **required** | **description**                                                 |
| ------------- | -------- | ------------ | --------------------------------------------------------------- |
| locationId    | string   | true         | The location ID (e.g. `us`, `us-central1`) of the Data Product. |
| dataProductId | string   | true         | The unique ID of the parent Data Product.                       |
| filter        | string   | false        | Filter string to list data assets.                              |
| pageSize      | integer  | false        | Number of returned data assets in the page.                     |
| orderBy       | string   | false        | Specifies the ordering of results.                              |

## Example

```yaml
kind: tool
name: list_data_assets
type: dataplex-list-data-assets
source: my-dataplex-source
description: Use this tool to list Data Assets under a Data Product.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
| ----------- | -------- | ------------ | -------------------------------------------------- |
| type        | string   | true         | Must be "dataplex-list-data-assets".               |
| source      | string   | true         | Name of the source the tool should execute on.     |
| description | string   | true         | Description of the tool that is passed to the LLM. |
