---
title: "dataplex-search-dq-scans"
type: docs
weight: 1
description: >
  A "dataplex-search-dq-scans" tool allows to search for data quality scans based on the provided parameters.
aliases:
- /resources/tools/dataplex-search-dq-scans
---

## About

A `dataplex-search-dq-scans` tool returns data quality scans that match the given criteria.
It's compatible with the following sources:

- [dataplex](../../sources/dataplex.md)

`dataplex-search-dq-scans` accepts the following optional parameters:

- `filter` - Filter string to search/filter data quality scans. E.g. "display_name = \"my-scan\"".
- `data_scan_id` - The ID of the data scan to filter by.
- `table_name` - The name of the table to filter by.
- `pageSize` - Number of returned data quality scans in the page. Defaults to `10`.
- `orderBy` - Specifies the ordering of results.

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
kind: tools
name: dataplex-search-dq-scans
type: dataplex-search-dq-scans
source: my-dataplex-source
description: Use this tool to search for data quality scans.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "dataplex-search-dq-scans".                |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
