---
title: "elasticsearch-esql"
type: docs
weight: 2
description: >
  Execute ES|QL queries.
---

# elasticsearch-esql

Execute ES|QL queries.

This tool allows you to execute ES|QL queries against your Elasticsearch
cluster. You can use this to perform complex searches and aggregations.

See the [official
documentation](https://www.elastic.co/docs/reference/query-languages/esql/esql-getting-started)
for more information.

## Example

```yaml
tools:
  query_my_index:
    kind: elasticsearch-esql
    source: elasticsearch-source
    description: Use this tool to execute ES|QL queries.
    query: |
      FROM my-index
      | KEEP *
      | LIMIT ?limit
    parameters:
      - name: limit
        type: integer
        description: Limit the number of results.
        required: true
```

## Parameters

| **name**   |                **type**                 | **required** | **description**                                                                                                                                     |
|------------|:---------------------------------------:|:------------:|-----------------------------------------------------------------------------------------------------------------------------------------------------|
| query      |                 string                  |    false     | The ES\|QL query to run. Can also be passed by parameters.                                                                                          |
| format     |                 string                  |    false     | The format of the query. Default is json. Valid values are csv, json, tsv, txt, yaml, cbor, smile, or arrow.                                        |
| timeout    |                 integer                 |    false     | The timeout for the query in seconds. Default is 60 (1 minute).                                                                                     |
| parameters | [parameters](../#specifying-parameters) |    false     | List of [parameters](../#specifying-parameters) that will be used with the ES\|QL query.<br/>Only supports “string”, “integer”, “float”, “boolean”. |
