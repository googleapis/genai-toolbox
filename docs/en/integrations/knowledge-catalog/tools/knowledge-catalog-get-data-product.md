---
title: "dataplex-get-data-product"
type: docs
weight: 1
description: >
  A "dataplex-get-data-product" tool allows to retrieve a specific Data Product.
---

## About

A `dataplex-get-data-product` tool retrieves detailed metadata for a specific Data Product in Knowledge Catalog (formerly known as Dataplex).

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

The `dataplex-get-data-product` tool has the following parameters:

| **field**     | **type** | **required** | **description**                                                 |
| ------------- | -------- | ------------ | --------------------------------------------------------------- |
| locationId    | string   | true         | The location ID (e.g. `us`, `us-central1`) of the Data Product. |
| dataProductId | string   | true         | The unique ID of the Data Product.                              |

## Example

```yaml
kind: tool
name: get_data_product
type: dataplex-get-data-product
source: my-dataplex-source
description: Use this tool to retrieve a Data Product.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
| ----------- | -------- | ------------ | -------------------------------------------------- |
| type        | string   | true         | Must be "dataplex-get-data-product".               |
| source      | string   | true         | Name of the source the tool should execute on.     |
| description | string   | true         | Description of the tool that is passed to the LLM. |
