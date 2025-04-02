---
title: "alloydb-ai-nl"
type: docs
weight: 1
description: > 
  A "alloydb-ai-nl" tool leverages AlloyDB's AI functions to execute natural language questions against the database.
---

## About

A `alloydb-ai-nl` tool leverages AlloyDB's AI functions to execute natural language questions against the database. It allows users to query database information using natural language instead of SQL. It's compatible with the following sources:
- [alloydb-postgres](../sources/alloydb-pg.md)

The tool uses AlloyDB's natural language processing capabilities to interpret questions and convert them into appropriate SQL queries, which are then executed against the database. TODO: link to AlloyDB's documentation.

## Fields

`nlConfig` is the name of the `nl_config` created in AlloyDB.

`nlConfigParameters` are the list of the parameters and values for the AlloyDB [PSV (parameterized secure views)](!https://cloud.google.com/alloydb/docs/ai/use-psvs#sanitize_queries_with_parameterized_secure_views).

When using this tool, all the PSV parameters should be from filled with values from an auth service or a bounded param. These parameters should not be visible to the LLM agent. Instead, the LLM will only see one argument when using this tool - `question`, with the description being "The natural language question to ask."

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
