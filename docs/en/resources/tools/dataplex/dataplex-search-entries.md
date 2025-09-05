---
title: "dataplex-search-entries"
type: docs
weight: 1
description: >
  A "dataplex-search-entries" tool allows to search for entries based on the provided query.
aliases:
- /resources/tools/dataplex-search-entries
---

## About

A `dataplex-search-entries` tool returns all entries in Dataplex Catalog (e.g.
tables, views, models) that matches given user query.
It's compatible with the following sources:

- [dataplex](../../sources/dataplex.md)

`dataplex-search-entries` takes a required `query` parameter based on which
entries are filtered and returned to the user. It also optionally accepts following parameters:

- `pageSize` - Number of results in the search page. Defaults to `5`.
- `orderBy` - Specifies the ordering of results. Supported values are: relevance
  (default), last_modified_timestamp, last_modified_timestamp asc.

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
  dataplex-search-entries:
    kind: dataplex-search-entries
    source: my-dataplex-source
    description: Use this tool to get all the entries based on the provided query.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| type        |                   string                   |     true     | Must be "dataplex-search-entries".                                                               |
| source      |                   string                   |     true     | Name of the source the tool should execute on.                                                   |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
