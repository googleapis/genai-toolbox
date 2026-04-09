---
title: "elasticsearch-execute-esql"
type: docs
weight: 3
description: >
  Execute arbitrary ES|QL statements.
---

## About

Execute arbitrary ES|QL statements.

This tool allows you to execute arbitrary ES|QL statements against your
Elasticsearch cluster at runtime. This is useful for ad-hoc queries where the
statement is not known beforehand.

See the
[official documentation](https://www.elastic.co/docs/reference/query-languages/esql/esql-getting-started)
for more information.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

| **name** | **type** | **required** | **description**                                                                                              |
| -------- | :------: | :----------: | ------------------------------------------------------------------------------------------------------------ |
| query    |  string  |     true     | The ES\|QL statement to execute.                                                                             |
| format   |  string  |    false     | The format of the query. Default is json. Valid values are csv, json, tsv, txt, yaml, cbor, smile, or arrow. |

## Example

```yaml
kind: tool
name: execute_ad_hoc_esql
type: elasticsearch-execute-esql
source: elasticsearch-source
description: Use this tool to execute arbitrary ES|QL statements.
format: json
```

