---
title: "dataplex-list-data-products"
type: docs
weight: 1
description: >
  A "dataplex-list-data-products" tool allows to list data products.
aliases:
  - /integrations/dataplex/tools/dataplex-list-data-products/
---

## About

A `dataplex-list-data-products` tool lists all Data Products in Knowledge Catalog (formerly known as Dataplex) across all locations (globally).

`dataplex-list-data-products` optionally accepts the following parameters:

- `filter` - Filter string to list data products. Use `=` for exact matching and `:` for contains matching. String literals must be enclosed within double quotes. E.g. `display_name:"my-product"`.
- `pageSize` - Number of returned data products in the page. Defaults to `10`.
- `orderBy` - Specifies the ordering of results.

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
name: list_data_products
type: dataplex-list-data-products
source: my-dataplex-source
description: Use this tool to list Data Products.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "dataplex-list-data-products".             |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
