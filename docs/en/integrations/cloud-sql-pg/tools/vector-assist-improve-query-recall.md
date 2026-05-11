---
title: "vector-assist-improve-query-recall"
type: docs
weight: 1
description: >
  The "vector-assist-improve-query-recall" tool generates SQL recommendations to improve search accuracy for users experiencing degraded recall.
---

## About

The `vector-assist-improve-query-recall` tool is designed to troubleshoot and optimize existing vector search workloads when a user reports irrelevant results, poor accuracy, or degraded recall. 

It determines the optimal tuning parameter (such as `hnsw.ef_search`) for an active vector index to improve the search results. The tool evaluates the target recall and outputs an actionable SQL query recommendation (e.g., `SET hnsw.ef_search TO ...`) that must be executed as a next action using the `execute_sql` tool.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

{{< notice tip >}} 
Ensure that your target PostgreSQL database has the required `vector_assist` extension installed. This tool requires an existing HNSW index to function properly; if the table lacks an existing vector setup, use the `define_spec` tool instead.
{{< /notice >}}

## Parameters

The tool takes the following input parameters:

| Parameter            | Type    | Description                                                                 | Required |
| :------------------- | :------ | :-------------------------------------------------------------------------- | :------- |
| `table_name`         | string  | Table name experiencing degraded vector search recall.                      | Yes      |
| `vector_column_name` | string  | Column name containing the vector embeddings.                               | Yes      |
| `index_name`         | string  | Name of the vector index to tune.                                           | Yes      |
| `schema_name`        | string  | Schema name of the table (default is `public`).                             | No       |
| `top_k`              | integer | Top k value for the vector search (default is `10`).                        | No       |
| `target_recall`      | float   | Target recall value for search results (default is `0.95`).                 | No       |
| `distance_func`      | string  | Distance function used for the vector search similarity (default is `cosine`).| No       |

> Note : 
> Parameters are marked as required or optional based on the tool's parameter definitions. 
> The underlying function may perform further validation on optional parameters to ensure 
> all necessary data is available before returning a response.

## Example

```yaml
kind: tool
name: improve_query_recall
type: vector-assist-improve-query-recall
source: my-database-source
description: "Use this tool to troubleshoot and optimize existing vector search workloads when a user reports irrelevant results, poor accuracy, or degraded recall. It determines the optimal tuning parameter (such as ef_search) for an active vector index to improve the search results. The tool outputs an actionable SQL query recommendation to be executed as a next action using the 'execute_sql' tool."
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| type        |  string  |     true     | Must be "vector-assist-improve-query-recall".        |
| source      |  string  |     true     | Name of the source the SQL should execute on.        |
| description |  string  |    false     | Description of the tool that is passed to the agent. |