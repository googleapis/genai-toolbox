---
title: "alloydb-ai-nl"
type: docs
weight: 1
description: > 
  The "alloydb-ai-nl" tool leverages AlloyDB's AI next-generation 
  [AI natural language]([alloydb-ai-nl-overview] support to provide the 
  ability to query the database directly using natural language.
---

## About

The `alloydb-ai-nl` tool leverages AlloyDB's next-generation AI natural 
language feature to allow an Agent the ability to query the database directly 
using natural language. Natural language streamlines the development of 
generative AI applications by transferring the complexity of converting 
natural language to SQL from the application layer to the database layer. 

This tool is compatible with the following sources:
- [alloydb-postgres](../sources/alloydb-pg.md)

AlloyDB AI natural language delivers secure and accurate responses for 
application end user natural language questions. Natural language streamlines 
the development of generative AI applications by transferring the complexity 
of converting natural language to SQL from the application layer to the 
database layer.

## Requirements
{{< notice tip >}}
AlloyDB AI natural language is currently in gated public preview. For more 
information on availability and limitations, please see 
[AlloyDB AI natural language overview][https://cloud.google.com/alloydb/docs/natural-language-questions-overview].
{{< /notice >}}

To enable AlloyDB AI natural language for your AlloyDB cluster, please follow 
the steps listed in the [Generate SQL queries that answer natural language questions][alloydb-ai-gen-nl], including enabling the extension and configuring context for your application.

[alloydb-ai-gen-nl]: https://cloud.google.com/alloydb/docs/alloydb/docs/ai/generate-queries-natural-language


## Configuration
### Configuration ID
`nlConfig` is the name of the `nl_config` created in AlloyDB. A `nl_config` 
is a configuration associates an application to schema objects, examples and 
other contexts that can be used. A large application can also use different 
configurations for different parts of the app, as long as the right 
configuration can be specified when a question is sent from that part of 
the application.


`nlConfigParameters` are the list of the parameters and values for the AlloyDB 
[Parameterized Secure Views (PSVs)][alloydb-psv].

When using this tool, we strongly recommend all the PSV parameters should be 
from filled with values from an auth service or a bounded param. These 
parameters should not be visible to the LLM agent.

[PSVs][alloydb-psv] 
are a feature unique to AlloyDB that allow you allow you to require one or 
more named parameter values passed to the view when querying it, somewhat 
like bind variables with ordinary database queries. You **must** supply 
all parameters required for all PSVs in the context. These parameters can be 
used with features like [Authenticated Parameters](../tools/#array-parameters) 
to provide secure access to queries generated using natural language.

[alloydb-psv]: https://cloud.google.com/alloydb/docs/ai/use-psvs#parameterized_secure_views

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

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "alloydb-ai-nl".                                                                          |
| source      |                   string                   |     true     | Name of the AlloyDB source the natural language query should execute on.                         |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
| nlConfig    |                   string                   |     true     | The name of the  `nl_config` in AlloyDB        |
| nlConfigParameters  | [parameters](_index#specifying-parameters) |    true     | List of PSV parameters defined in the `nl_config`  |
