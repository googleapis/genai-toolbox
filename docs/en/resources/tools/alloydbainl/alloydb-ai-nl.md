---
title: "alloydb-ai-nl"
type: docs
weight: 1
description: >
  The "alloydb-ai-nl" tool leverages
  [AlloyDB AI](https://cloud.google.com/alloydb/ai) next-generation Natural
  Language support to provide the ability to query the database directly using
  natural language.
aliases:
- /resources/tools/alloydb-ai-nl
---

## About

The `alloydb-ai-nl` tool leverages [AlloyDB AI next-generation natural
Language][alloydb-ai-nl-overview] support to allow an Agent the ability to query
the database directly using natural language. Natural language streamlines the
development of generative AI applications by transferring the complexity of
converting natural language to SQL from the application layer to the database
layer.

This tool is compatible with the following sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)

AlloyDB AI Natural Language delivers secure and accurate responses for
application end user natural language questions. Natural language streamlines
the development of generative AI applications by transferring the complexity
of converting natural language to SQL from the application layer to the
database layer.

## Requirements

{{< notice tip >}} AlloyDB AI natural language is currently in gated public
preview. For more information on availability and limitations, please see
[AlloyDB AI natural language overview](https://cloud.google.com/alloydb/docs/ai/natural-language-overview)
{{< /notice >}}

To enable AlloyDB AI natural language for your AlloyDB cluster, please follow
the steps listed in the [Generate SQL queries that answer natural language
questions][alloydb-ai-gen-nl], including enabling the extension and configuring
context for your application.

[alloydb-ai-nl-overview]: https://cloud.google.com/alloydb/docs/ai/natural-language-overview
[alloydb-ai-gen-nl]: https://cloud.google.com/alloydb/docs/ai/generate-sql-queries-natural-language

## Configuration

### Specifying an `nl_config`

A `nl_config` is a configuration that associates an application to schema
objects, examples and other contexts that can be used. A large application can
also use different configurations for different parts of the app, as long as the
correct configuration can be specified when a question is sent from that part of
the application.

Once you've followed the steps for configuring context, you can use the
`context` field when configuring a `alloydb-ai-nl` tool. When this tool is
invoked, the SQL will be generated and executed using this context.

### Specifying Parameters to PSV's

[Parameterized Secure Views (PSVs)][alloydb-psv] are a feature unique to AlloyDB
that allows you to require one or more named parameter values passed
to the view when querying it, somewhat like bind variables with ordinary
database queries.

You can use the `nlConfigParameters` to list the parameters required for your
`nl_config`. You **must** supply all parameters required for all PSVs in the
context. It's strongly recommended to use features like [Authenticated
Parameters](../#array-parameters) or Bound Parameters to provide secure
access to queries generated using natural language, as these parameters are not
visible to the LLM.

[alloydb-psv]: https://cloud.google.com/alloydb/docs/parameterized-secure-views-overview

{{< notice tip >}} Make sure to enable the `parameterized_views` extension before running this tool. You can do so by running this command in the AlloyDB studio:
```sql
CREATE EXTENSION IF NOT EXISTS parameterized_views;
```
{{< /notice >}}

## Example

```yaml
tools:
  ask_questions:
    kind: alloydb-ai-nl
    source: my-alloydb-source
    description: "Ask questions to check information about flights"
    nlConfig: "cymbal_air_nl_config"
    nlConfigParameters:
      - name: user_email
        type: string
        description: User ID of the logged in user.
        # note: we strongly recommend using features like Authenticated or
        # Bound parameters to prevent the LLM from seeing these params and
        # specifying values it shouldn't in the tool input
        authServices:
          - name: my_google_service
            field: email
```
## Reference

| **field**          |                  **type**                  | **required** | **description**                                                          |
|--------------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------|
| type               |                   string                   |     true     | Must be "alloydb-ai-nl".                                                 |
| source             |                   string                   |     true     | Name of the AlloyDB source the natural language query should execute on. |
| description        |                   string                   |     true     | Description of the tool that is passed to the LLM.                       |
| nlConfig           |                   string                   |     true     | The name of the  `nl_config` in AlloyDB                                  |
| nlConfigParameters | [parameters](../#specifying-parameters) |     true     | List of PSV parameters defined in the `nl_config`                        |
