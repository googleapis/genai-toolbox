---
title: "dataplex-lookup-entry"
type: docs
weight: 1
description: > 
  A "dataplex-lookup-entry" tool returns details of a particular entry in Dataplex Catalog.
aliases:
- /resources/tools/dataplex-lookup-entry
---

## About

A `dataplex-lookup-entry` tool returns details of a particular entry in Dataplex
Catalog. It's compatible with the following sources:

- [dataplex](../../sources/dataplex.md)

`dataplex-lookup-entry` takes a required `name` parameter which contains the
project and location to which the request should be attributed in the following
form: projects/{project}/locations/{location} and also a required `entry`
parameter which is the resource name of the entry in the following form:
projects/{project}/locations/{location}/entryGroups/{entryGroup}/entries/{entry}.
It also optionally accepts following parameters:
  - `view` - View to control which parts of an entry the service should return.
    It takes integer values from 1-4 corresponding to type of view - BASIC,
    FULL, CUSTOM, ALL
  - `aspectTypes` - Limits the aspects returned to the provided aspect types in
    the format
    `projects/{project}/locations/{location}/aspectTypes/{aspectType}`. It only
    works for CUSTOM view.
  - `paths` - Limits the aspects returned to those associated with the provided
    paths within the Entry. It only works for CUSTOM view.

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

## Example

```yaml
tools:
  lookup_entry:
    kind: dataplex-lookup-entry
    source: my-dataplex-source
    description: Use this tool to retrieve a specific entry in Dataplex Catalog.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "dataplex-lookup-entry".                   |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |