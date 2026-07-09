---
title: "dataplex-list-data-products"
type: docs
weight: 1
description: >
  A "dataplex-list-data-products" tool allows to list data products.
---

## About

A `dataplex-list-data-products` tool lists all Data Products in Knowledge Catalog (formerly known as Dataplex) across all locations (globally).

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

The `dataplex-list-data-products` tool has the following optional parameters:

| **field** | **type** | **required** | **description**                                                                                                                                                                                                                                                |
| --------- | -------- | ------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| filter    | string   | false        | Filter string to list data products. Based on the AIP-160 proposal. Use '=' for exact, and ':' for contains matching. String literals must be enclosed within "". Matching across all fields at once is not yet supported. E.g. "display_name:\"my-product\"" |
| pageSize  | integer  | false        | Number of returned data products in the page. Defaults to `10`.                                                                                                                                                                                                |
| orderBy   | string   | false        | Specifies the ordering of results.                                                                                                                                                                                                                             |

## Example

```yaml
kind: tool
name: list_data_products
type: dataplex-list-data-products
source: my-dataplex-source
description: Use this tool to list Data Products.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
| ----------- | -------- | ------------ | -------------------------------------------------- |
| type        | string   | true         | Must be "dataplex-list-data-products".             |
| source      | string   | true         | Name of the source the tool should execute on.     |
| description | string   | true         | Description of the tool that is passed to the LLM. |
