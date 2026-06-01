---
title: "vector-assist-generate-query"
type: docs
weight: 1
description: >
  The "vector-assist-generate-query" tool produces optimized SQL queries for
  vector search, leveraging metadata and specifications to enable semantic
  and similarity searches.
---

## About

The `vector-assist-generate-query` tool generates optimized SQL queries for vector search by leveraging the metadata and vector specifications defined in a specific spec_id. It serves as the primary actionable tool for generating the executable SQL required to retrieve relevant results based on vector similarity.

The tool contextually understands requirements such as distance functions, quantization, and filtering to ensure the resulting query is compatible with the corresponding vector index. Additionally, it can automatically handle iterative index scans for filtered queries and calculate the necessary search parameters (like ef_search) to meet a target recall.
## Compatible Sources

{{< compatible-sources >}}

## Requirements

{{< notice tip >}} 
Ensure that your target PostgreSQL database has the required `vector_assist` extension installed, in order for this tool to execute successfully.
{{< /notice >}}

## Parameters

The tool takes the following input parameters:

| Parameter                | Type    | Description                                                         | Required |
| :----------------------- | :------ | :------------------------------------------------------------------ | :------- |
| `spec_id`                | string  | Unique ID of the vector spec for query generation.                  | No       |
| `table_name`             | string  | Target table name for generating the vector query.                  | No       |
| `schema_name`            | string  | Schema name for the query's target table.                           | No       |
| `column_name`            | string  | Text or vector column name identifying the specific spec.           | No       |
| `search_text`            | string  | Text string to search for; embeddings are auto-generated.           | No       |
| `search_vector`          | string  | Vector to search for; use instead of search_text.                   | No       |
| `output_column_names`    | array   | List of columns to retrieve in the search results.                  | No       |
| `top_k`                  | integer | Number of nearest neighbors to return (defaults to 10).             | No       |
| `filter_expressions`     | array   | List of filter expressions applied to the vector query.             | No       |
| `target_recall`          | float   | Target recall rate, overriding the spec-level default.              | No       |
| `iterative_index_search` | boolean | Enables iterative search for filtered queries to guarantee results. | No       |

> Note
> Parameters are marked as required or optional based on the vector assist function definitions. 
> The function may perform further validation on optional parameters to ensure all necessary 
> data is available before returning a response.

## Example

```yaml
kind: tool
name: generate_query
type: vector-assist-generate-query
source: my-database-source
description: "This tool generates optimized SQL queries for vector search by leveraging the metadata and vector specifications defined in a specific spec_id. It may return a single query or a sequence of multiple SQL queries that can be executed sequentially. Use this tool when a user wants to perform semantic or similarity searches on their data. It serves as the primary actionable tool to invoke for generating the executable SQL required to retrieve relevant results based on vector similarity."
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| type        |  string  |     true     | Must be "vector-assist-generate-query".                 |
| source      |  string  |     true     | Name of the source the SQL should execute on.        |
| description |  string  |    false     | Description of the tool that is passed to the agent. |