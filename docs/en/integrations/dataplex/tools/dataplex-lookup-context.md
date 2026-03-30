---
title: "dataplex-lookup-context"
type: docs
weight: 1
description: > 
  A "dataplex-lookup-context" tool provides rich metadata of one or more data assets along with their relationships.
---

## About

A `dataplex-lookup-context` tool provides rich metadata of one or more data assets along with their relationships.

`dataplex-lookup-context` takes a required `name` parameter which contains the
project and location to which the request should be attributed in the following
form: projects/{project}/locations/{location} and also a required `resources`
parameter which is a list of resource names for which metadata is needed in the 
following form: projects/{project}/locations/{location}/entryGroups/{group}/entries/{entry}

## Compatible Sources

{{< compatible-sources >}}

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

**Note on Lookup Context Tool Behavior:** This specific tool utilizes a post-filtering
approach for authorization. This means that any authenticated user can call the tool's
API endpoint. However, the response will only contain data for resources that the
caller's identity (via ADC) has the necessary IAM permissions to access. If the caller
has no permissions on the requested resources, the tool will return an empty response
rather than an access denied error.

[iam-overview]: https://cloud.google.com/dataplex/docs/iam-and-access-control
[adc]: https://cloud.google.com/docs/authentication#adc
[set-adc]: https://cloud.google.com/docs/authentication/provide-credentials-adc
[iam-permissions]: https://cloud.google.com/dataplex/docs/iam-permissions
[iam-roles]: https://cloud.google.com/dataplex/docs/iam-roles
[dataplex-docs]: https://cloud.google.com/dataplex/docs

## Example

```yaml
kind: tool
name: lookup_context
type: dataplex-lookup-context
source: my-dataplex-source
description: Use this tool to retrieve rich metadata regarding one or more data assets along with their relationships.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "dataplex-lookup-context".                 |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
