---
title: "firestoremongodb-execute-mql"
type: docs
weight: 1
description: >
  A "firestoremongodb-execute-mql" tool executes MongoDB Query Language (MQL) queries and aggregation pipelines against Firestore.
---

## About

A `firestoremongodb-execute-mql` tool executes MongoDB Query Language (MQL) statements or aggregation pipelines against Firestore.

This tool allows data agents and LLMs to run dynamic MQL queries such as aggregation pipelines or document queries directly against Firestore databases.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | true | The MQL query or aggregation pipeline to execute against Firestore. |

## Example

```yaml
kind: tool
name: execute_mql
type: firestoremongodb-execute-mql
source: my-firestore-source
description: Use this tool to execute MQL queries against Firestore.
```

## Reference

| **field**   | **type** | **required** | **description** |
|-------------|:--------:|:------------:|-----------------|
| type | string | true | Must be "firestoremongodb-execute-mql". |
| source | string | true | Name of the Firestore source to execute queries against. |
| description | string | true | Description of the tool that is passed to the LLM. |
