---
title: "dataplex-lookup-entry"
type: docs
weight: 1
description: >
  A "dataplex-lookup-entry" tool returns details of a particular entry in Knowledge Catalog.
aliases:
  - /integrations/dataplex/tools/dataplex-lookup-entry/
---

## About

A `dataplex-lookup-entry` tool returns details of a particular entry in Knowledge Catalog.

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

## Parameters

The `dataplex-lookup-entry` tool accepts the following parameters:

| **field** | **type** | **required** | **description** |
| --------- | :------: | :----------: | --------------- |
| entry | string | true | The resource name of the Entry (format: `projects/{project}/locations/{location}/entryGroups/{entryGroup}/entries/{entry}`). |
| view | integer | false | View to control which parts of an entry to return (1=BASIC, 2=FULL, 3=CUSTOM, 4=ALL). Defaults to 2. |
| aspectTypes | list of strings | false | Limits the aspects returned to the provided aspect types (format: `projects/{project}/locations/{location}/aspectTypes/{aspectType}`). Only works for CUSTOM view (3). |

## Example

```yaml
kind: tool
name: lookup_entry
type: dataplex-lookup-entry
source: my-dataplex-source
description: Use this tool to retrieve a specific entry in Knowledge Catalog.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "dataplex-lookup-entry".                   |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
