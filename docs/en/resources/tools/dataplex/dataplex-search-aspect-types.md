---
title: "dataplex-search-aspect-types"
type: docs
weight: 1
description: >
  A "dataplex-search-aspect-types" tool allows to fetch the metadata template of 3P aspect types based on search query.
aliases:
- /resources/tools/dataplex-search-aspect-types
---

## About

A `dataplex-search-aspect-types` tool allows to fetch the metadata template of 3P aspect types based on search query.
It's compatible with the following sources:

- [dataplex](../../sources/dataplex.md)

`dataplex-search-aspect-types` accepts following parameters optionally:

- `query` - Narrows down the search of aspect types to value of this parameter. If not provided, it fetches all 3P aspect types available to the user.
- `pageSize` - Number of results in the search page. Defaults to `5`.
- `pageToken` - Page token received from a previous search aspect types tool call.
- `orderBy` - Specifies the ordering of results. Supported values are: relevance (default), last_modified_timestamp, last_modified_timestamp asc
- `semanticSearch` - Specifies whether the aspect type search should understand the meaning and intent behind the query, rather than just matching keywords. Defaults to `true`.
- `scope` - The scope under which the search should be operating. Since this parameter is not exposed to the toolbox user, it defaults to the organization where the project provided in name is located.

## Requirements

### IAM Permissions

Dataplex uses [Identity and Access Management (IAM)][iam-overview] to control
user and group access to Dataplex resources. Toolbox will use your
[Application Default Credentials (ADC)][adc] to authorize and authenticate when
interacting with [Dataplex][dataplex-docs].

In addition to [setting the ADC for your server][set-adc], you need to ensure
the IAM identity has been given the correct IAM permissions for the tasks you
intend to perform. See [Dataplex Universal Catalog IAM permissions][iam-permissions]
and [Dataplex Universal Catalog IAM roles][iam-roles] for more information on
applying IAM permissions and roles to an identity.

[iam-overview]: https://cloud.google.com/dataplex/docs/iam-and-access-control
[adc]: https://cloud.google.com/docs/authentication#adc
[set-adc]: https://cloud.google.com/docs/authentication/provide-credentials-adc
[iam-permissions]: https://cloud.google.com/dataplex/docs/iam-permissions
[iam-roles]: https://cloud.google.com/dataplex/docs/iam-roles
[dataplex-docs]: https://cloud.google.com/dataplex

## Example

```yaml
tools:
  dataplex-search-aspect-types:
    kind: dataplex-search-aspect-types
    source: my-dataplex-source
    description: Use this tool to find aspect types relevant to the query.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "dataplex-search-aspect-types".                                                          |
| source      |                   string                   |     true     | Name of the source the tool should execute on.                                                   |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |