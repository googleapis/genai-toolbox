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

The "alloydb-ai-nl" tool leverages AlloyDB's next-generation AI natural 
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

## Fields

`nlConfig` is the name of the `nl_config` created in AlloyDB.

`nlConfigParameters` are the list of the parameters and values for the AlloyDB 
[PSV (parameterized secure views)](!https://cloud.google.com/alloydb/docs/ai/use-psvs#sanitize_queries_with_parameterized_secure_views).

When using this tool, all the PSV parameters should be from filled with values 
from an auth service or a bounded param. These parameters should not be 
visible to the LLM agent. Instead, the LLM will only see one argument when 
using this tool - `question`, with the description being "The natural 
language question to ask."

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
| nlConfig   |                   string                   |     true     | The name of the  `nl_config` in AlloyDB        |
| nlConfigParameters  | [parameters](_index#specifying-parameters) |    true     | List of PSV parameters defined in the `nl_config`  |
