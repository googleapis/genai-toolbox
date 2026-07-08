---
title: "dataplex-create-data-asset"
type: docs
weight: 2
description: >
  A "dataplex-create-data-asset" tool creates a new Data Asset under an existing Data Product in Knowledge Catalog.
---

## About

A `dataplex-create-data-asset` tool creates a new Data Asset under a Data Product in Knowledge Catalog (formerly known as Dataplex). This is a long-running operation, and the tool returns immediately with the operation's location ID and operation ID.

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

The `dataplex-create-data-asset` tool accepts the following parameters:

| **field**          | **type** | **required** | **description**                                                                                                                                                                                                                                                                                                                                                             |
| ------------------ | -------- | ------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| locationId         | string   | true         | The location ID (e.g. `us`, `us-central1`) where the parent Data Product is located.                                                                                                                                                                                                                                                                                        |
| dataProductId      | string   | true         | The unique ID of the parent Data Product.                                                                                                                                                                                                                                                                                                                                   |
| dataAssetId        | string   | true         | The unique ID of the Data Asset to create.                                                                                                                                                                                                                                                                                                                                  |
| resourceType       | string   | true         | The type of resource to be represented by this data asset. Supported values are: `BIGQUERY_DATASET`, `BIGQUERY_TABLE`, `BIGQUERY_VIEW`, `BIGQUERY_ROUTINE`, `BIGQUERY_MODEL`, `GEMINI_ENTERPRISE_AGENT_PLATFORM_DATASET`, `GEMINI_ENTERPRISE_AGENT_PLATFORM_MODEL`, `CLOUD_STORAGE_BUCKET`.                                                                                 |
| resourceProjectId  | string   | false        | The project ID of the resource. This is NOT required if the resource is a `CLOUD_STORAGE_BUCKET`.                                                                                                                                                                                                                                                                 |
| resourceLocationId | string   | false        | The location of the resource. This is required if the resource type is `GEMINI_ENTERPRISE_AGENT_PLATFORM_DATASET` or `GEMINI_ENTERPRISE_AGENT_PLATFORM_MODEL`.                                                                                                                                                                                                    |
| resourceDatasetId  | string   | false        | The dataset ID of the resource. This is required if the resource type is `BIGQUERY_*` or `GEMINI_ENTERPRISE_AGENT_PLATFORM_DATASET`.                                                                                                                                                                                                                              |
| resourceId         | string   | false        | The resource ID of the resource.                                                                                                                                                                                                                                                                                                                                  |
| labels             | map      | false        | The labels associated with the Data Asset. Keys and values must be strings.                                                                                                                                                                                                                                                                                                 |
| accessGroupConfigs | map      | false        | Map of access group configurations to associate with the Data Asset. Keys represent the access group ID, and the value is a list of string IAM role names (e.g. `{"test-group": ["roles/bigquery.dataViewer"]}`). To find the list of supported roles that can be granted on the resource, refer to the [roles:queryGrantableRoles][query-grantable-roles-docs] API method. |

[query-grantable-roles-docs]: https://cloud.google.com/iam/docs/reference/rest/v1/roles/queryGrantableRoles

## Example

```yaml
kind: tool
name: create_data_asset
type: dataplex-create-data-asset
source: my-dataplex-source
description: Use this tool to create a Data Asset.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
| ----------- | -------- | ------------ | -------------------------------------------------- |
| type        | string   | true         | Must be "dataplex-create-data-asset".              |
| source      | string   | true         | Name of the source the tool should execute on.     |
| description | string   | true         | Description of the tool that is passed to the LLM. |
