---
title: "dataplex-create-data-product"
type: docs
weight: 2
description: >
  A "dataplex-create-data-product" tool allows to create a new Data Product.
aliases:
  - /integrations/dataplex/tools/dataplex-create-data-product/
---

## About

A `dataplex-create-data-product` tool creates a new Data Product in Knowledge Catalog (formerly known as Dataplex). This is a long-running operation, and the tool returns immediately with the operation's resource name.

`dataplex-create-data-product` accepts the following parameters:

- `name` - Required. The resource name of the Data Product in the format `projects/{project}/locations/{location}/dataProducts/{dataProduct}`.
- `displayName` - Required. The display name of the Data Product.
- `description` - Optional. The description of the Data Product.
- `ownerEmails` - Required. The list of owner emails for the Data Product.
- `accessGroups` - Optional. List of access groups to associate with the Data Product. Each group object can contain: `id` (required), `displayName` (required), `description`, and at least one of `googleGroup` and `serviceAccount`.

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
name: create_data_product
type: dataplex-create-data-product
source: my-dataplex-source
description: Use this tool to create a Data Product.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "dataplex-create-data-product".            |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
