---
title: "postgres-search-catalog"
type: docs
weight: 1
description: >
  A "postgres-search-catalog" tool allows to search for entries based on the provided query.
---

## About

A `postgres-search-catalog` tool returns all entries in Knowledge Catalog (e.g.
tables, views, databases) with system=PostgreSQL that matches given user query.

`postgres-search-catalog` takes a required `prompt` parameter based on which
entries are filtered and returned to the user. It also optionally accepts
following parameters:

- `databaseIds` - The IDs of the postgres database.
- `projectIds` - The IDs of the GCP project.
- `types` - The type of the data. Accepted values are: DATABASE, TABLE, VIEW.
- `pageSize` - Number of results in the search page. Defaults to `5`.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

### IAM Permissions

Postgres uses [Identity and Access Management (IAM)][iam-overview] to control
user and group access to Knowledge Catalog (formerly known as Dataplex) resources. Toolbox will use your
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
[dataplex-docs]: https://cloud.google.com/dataplex/docs

## Example

```yaml
kind: tool
name: search_catalog
type: postgres-search-catalog
source: cloudsql-pg-source
description: Searches for data assets (eg. Postgres tables, views, or databases) in catalog based on the provided search query
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| type        |                   string                   |     true     | Must be "postgres-search-catalog".                                                               |
| source      |                   string                   |     true     | Name of the source the tool should execute on.                                                   |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
