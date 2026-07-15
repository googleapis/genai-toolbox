---
title: "dataplex-search-aspect-types"
type: docs
weight: 1
description: >
  A "dataplex-search-aspect-types" tool allows to find aspect types relevant to the query.
aliases:
  - /integrations/dataplex/tools/dataplex-search-aspect-types/
---

## About

A `dataplex-search-aspect-types` tool allows to fetch the metadata template of
aspect types based on search query.

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

The `dataplex-search-aspect-types` tool accepts the following parameters:

| **field** | **type** | **required** | **description** |
| --------- | :------: | :----------: | --------------- |
| query | string | false | Optional. Narrows down the search of aspect types to value of this parameter. If not provided, it fetches all available aspect types. |
| pageSize | integer | false | Number of returned aspect types in the search page. Defaults to 5. |
| orderBy | string | false | Specifies ordering of results (`relevance`, `last_modified_timestamp`, `last_modified_timestamp asc`). Defaults to relevance. |

## Example

```yaml
kind: tool
name: search_aspect_types
type: dataplex-search-aspect-types
source: my-dataplex-source
description: Use this tool to find aspect types relevant to the query.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "dataplex-search-aspect-types".            |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
